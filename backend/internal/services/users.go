package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Services) GetMe(ctx context.Context, claims *middleware.UserClaims) (*domain.UserPrivate, error) {
	var u struct {
		ID                 string
		ScreenName         string
		Email              string
		Role               string
		Bio                pgtype.Text
		AvatarURL          pgtype.Text
		Locale             pgtype.Text
		NotifyPreferences  []byte // jsonb
		EmailVerifiedAt    pgtype.Timestamptz
		AcceptTermsVersion pgtype.Text
		CreatedAt          pgtype.Timestamptz
		UpdatedAt          pgtype.Timestamptz
	}

	err := s.DB.QueryRow(ctx,
		`SELECT id, screen_name, email, role, bio, avatar_url,
		        locale, notify_preferences,
		        email_verified_at, accept_terms_version,
		        created_at, updated_at
		 FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&u.ID, &u.ScreenName, &u.Email, &u.Role, &u.Bio, &u.AvatarURL,
		&u.Locale, &u.NotifyPreferences,
		&u.EmailVerifiedAt, &u.AcceptTermsVersion,
		&u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	locale := "pt-BR"
	if u.Locale.Valid {
		locale = u.Locale.String
	}

	var notifyPrefs map[string]any
	if len(u.NotifyPreferences) > 0 {
		if err := json.Unmarshal(u.NotifyPreferences, &notifyPrefs); err != nil {
			if s.Logger != nil {
				s.Logger.Error("malformed notify_preferences in DB", "user_id", u.ID, "error", err)
			}
		}
	}
	if notifyPrefs == nil {
		notifyPrefs = map[string]any{"email": true, "push": true}
	}

	var emailVerifiedAt *string
	if u.EmailVerifiedAt.Valid {
		s := utils.FormatRFC3339UTC(u.EmailVerifiedAt.Time)
		emailVerifiedAt = &s
	}

	return &domain.UserPrivate{
		ID:                 u.ID,
		ScreenName:         u.ScreenName,
		Email:              u.Email,
		Role:               u.Role,
		Bio:                u.Bio.String,
		AvatarURL:          u.AvatarURL.String,
		Locale:             locale,
		NotifyPreferences:  notifyPrefs,
		EmailVerifiedAt:    emailVerifiedAt,
		AcceptTermsVersion: u.AcceptTermsVersion.String,
		CreatedAt:          utils.FormatRFC3339UTC(u.CreatedAt.Time),
		UpdatedAt:          utils.FormatRFC3339UTC(u.UpdatedAt.Time),
	}, nil
}

func (s *Services) UpdateMe(ctx context.Context, claims *middleware.UserClaims, req domain.PatchMeRequest) (*domain.UserPrivate, error) {
	if errs, ok := req.Validate(); !ok {
		return nil, domain.ErrValidation("Validation failed", errs)
	}

	// Build dynamic SET clause from non-nil whitelisted fields
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.ScreenName != nil {
		setClauses = append(setClauses, fmt.Sprintf("screen_name = $%d", argIdx))
		args = append(args, *req.ScreenName)
		argIdx++
	}
	if req.Bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio = $%d", argIdx))
		args = append(args, *req.Bio)
		argIdx++
	}
	if req.Locale != nil {
		setClauses = append(setClauses, fmt.Sprintf("locale = $%d", argIdx))
		args = append(args, *req.Locale)
		argIdx++
	}
	if req.NotifyPreferences != nil {
		jsonBytes, err := json.Marshal(*req.NotifyPreferences)
		if err != nil {
			return nil, domain.Err(400, "Invalid notify_preferences")
		}
		setClauses = append(setClauses, fmt.Sprintf("notify_preferences = $%d", argIdx))
		args = append(args, jsonBytes)
		argIdx++
	}

	if len(setClauses) == 0 {
		// No fields to update — just return current profile
		return s.GetMe(ctx, claims)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, claims.UserID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := s.DB.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.Err(409, "Screen name already taken")
		}
		return nil, domain.Err(500, "Failed to update user")
	}

	return s.GetMe(ctx, claims)
}

func (s *Services) DeleteMe(ctx context.Context, claims *middleware.UserClaims) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE users SET deleted_at = NOW(), email = CONCAT('deleted_', id, '@example.com') WHERE id = $1`,
		claims.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to delete user")
	}
	return nil
}

func (s *Services) GetUserByID(ctx context.Context, id string) (*domain.UserPublic, error) {
	var u struct {
		ID         string
		ScreenName string
		Role       string
		Bio        pgtype.Text
		AvatarURL  pgtype.Text
		CreatedAt  pgtype.Timestamptz
	}

	err := s.DB.QueryRow(ctx,
		`SELECT id, screen_name, role, bio, avatar_url, created_at FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.ScreenName, &u.Role, &u.Bio, &u.AvatarURL, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "User not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	return &domain.UserPublic{
		ID:         u.ID,
		ScreenName: u.ScreenName,
		Role:       u.Role,
		Bio:        u.Bio.String,
		AvatarURL:  u.AvatarURL.String,
		CreatedAt:  utils.FormatRFC3339UTC(u.CreatedAt.Time),
	}, nil
}

func (s *Services) ListUsers(ctx context.Context, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id, screen_name, role, avatar_url, created_at FROM users 
		 WHERE deleted_at IS NULL AND ($1 = '' OR created_at < $1)
		 ORDER BY created_at DESC LIMIT $2`,
		cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var users []any
	for rows.Next() {
		var u domain.UserPublic
		err := rows.Scan(&u.ID, &u.ScreenName, &u.Role, &u.AvatarURL, &u.CreatedAt)
		if err != nil {
			continue
		}
		users = append(users, u)
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}

	nextCursor := ""
	if hasMore && len(users) > 0 {
		nextCursor = users[len(users)-1].(domain.UserPublic).CreatedAt
	}

	return &domain.CursorPage{Data: users, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Services) GetUserPosts(ctx context.Context, userID, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.screen_name, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.author_id = $1 AND p.deleted_at IS NULL AND p.visibility = 'public'
		 AND ($2 = '' OR p.created_at < $2)
		 ORDER BY p.created_at DESC LIMIT $3`,
		userID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var posts []any
	for rows.Next() {
		var p domain.PostResponse
		var author domain.UserPublic
		err := rows.Scan(&p.ID, &p.AuthorID, &p.Content, &p.MediaURLs, &p.Visibility, &p.LikeCount,
			&p.CreatedAt, &p.UpdatedAt, &author.ID, &author.ScreenName, &author.Role, &author.AvatarURL)
		if err != nil {
			continue
		}
		p.Author = author
		posts = append(posts, p)
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	nextCursor := ""
	if hasMore && len(posts) > 0 {
		nextCursor = posts[len(posts)-1].(domain.PostResponse).CreatedAt
	}

	return &domain.CursorPage{Data: posts, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Services) SearchUsers(ctx context.Context, query, cursor string) (*domain.CursorPage, error) {
	if query == "" {
		return nil, domain.ErrValidation("Search query is required", map[string]string{"q": "Search query is required"})
	}

	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id, screen_name, role, avatar_url, created_at FROM users 
		 WHERE deleted_at IS NULL
		 AND (to_tsvector('portuguese', coalesce(screen_name,'') || ' ' || coalesce(bio,'')) @@ plainto_tsquery('portuguese', $1))
		 AND ($2 = '' OR created_at < $2)
		 ORDER BY created_at DESC LIMIT $3`,
		query, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var users []any
	for rows.Next() {
		var u domain.UserPublic
		err := rows.Scan(&u.ID, &u.ScreenName, &u.Role, &u.AvatarURL, &u.CreatedAt)
		if err != nil {
			continue
		}
		users = append(users, u)
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}

	nextCursor := ""
	if hasMore && len(users) > 0 {
		nextCursor = users[len(users)-1].(domain.UserPublic).ID
	}

	return &domain.CursorPage{Data: users, NextCursor: nextCursor, HasMore: hasMore}, nil
}
