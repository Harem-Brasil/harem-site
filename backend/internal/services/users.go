package services

import (
	"context"
	"errors"

	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Services) GetMe(ctx context.Context, claims *middleware.UserClaims) (*domain.UserPublic, error) {
	var u struct {
		ID        string
		Username  string
		Email     string
		Role      string
		Bio       pgtype.Text
		AvatarURL pgtype.Text
		CreatedAt pgtype.Timestamptz
	}

	err := s.DB.QueryRow(ctx,
		`SELECT id, username, email, role, bio, avatar_url, created_at FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Bio, &u.AvatarURL, &u.CreatedAt)

	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	return &domain.UserPublic{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Bio:       u.Bio.String,
		AvatarURL: u.AvatarURL.String,
		CreatedAt: utils.FormatRFC3339UTC(u.CreatedAt.Time),
	}, nil
}

func (s *Services) UpdateMe(ctx context.Context, claims *middleware.UserClaims, updates map[string]any) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE users SET username = COALESCE($1, username), bio = COALESCE($2, bio), updated_at = NOW() WHERE id = $3`,
		updates["username"], updates["bio"], claims.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to update user")
	}
	return nil
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
		ID        string
		Username  string
		Role      string
		Bio       pgtype.Text
		AvatarURL pgtype.Text
		CreatedAt pgtype.Timestamptz
	}

	err := s.DB.QueryRow(ctx,
		`SELECT id, username, role, bio, avatar_url, created_at FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Bio, &u.AvatarURL, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "User not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	return &domain.UserPublic{
		ID:        u.ID,
		Username:  u.Username,
		Role:      u.Role,
		Bio:       u.Bio.String,
		AvatarURL: u.AvatarURL.String,
		CreatedAt: utils.FormatRFC3339UTC(u.CreatedAt.Time),
	}, nil
}

func (s *Services) ListUsers(ctx context.Context, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id, username, role, avatar_url, created_at FROM users 
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
		err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.AvatarURL, &u.CreatedAt)
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
		        u.id, u.username, u.role, u.avatar_url
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
			&p.CreatedAt, &p.UpdatedAt, &author.ID, &author.Username, &author.Role, &author.AvatarURL)
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
		`SELECT id, username, role, avatar_url, created_at FROM users 
		 WHERE deleted_at IS NULL
		 AND (to_tsvector('portuguese', coalesce(username,'') || ' ' || coalesce(bio,'')) @@ plainto_tsquery('portuguese', $1))
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
		err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.AvatarURL, &u.CreatedAt)
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
