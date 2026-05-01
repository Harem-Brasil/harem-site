package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
)

func (s *Services) PostCreatorApply(ctx context.Context, user *middleware.UserClaims, bio string, socialLinks []string) (*domain.CreatorApplication, error) {
	var existingID string
	err := s.DB.QueryRow(ctx,
		`SELECT id FROM creator_applications WHERE user_id = $1 LIMIT 1`,
		user.UserID,
	).Scan(&existingID)
	if err == nil {
		if s.Logger != nil {
			s.Logger.Info("creator apply rejected: already submitted", "user_id", user.UserID)
		}
		return nil, domain.Err(http.StatusConflict, "Creator application already submitted")
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(500, "Database error")
	}

	appID := uuid.New().String()
	now := time.Now().UTC()

	links := socialLinks
	if links == nil {
		links = []string{}
	}
	socialLinksJSON, err := json.Marshal(links)
	if err != nil {
		return nil, domain.Err(500, "Failed to submit application")
	}

	_, err = s.DB.Exec(ctx,
		`INSERT INTO creator_applications (id, user_id, bio, social_links, status, submitted_at) 
		 VALUES ($1, $2, $3, $4::jsonb, 'pending', $5)`,
		appID, user.UserID, bio, socialLinksJSON, now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if s.Logger != nil {
				s.Logger.Info("creator apply rejected: duplicate insert race", "user_id", user.UserID)
			}
			return nil, domain.Err(http.StatusConflict, "Creator application already submitted")
		}
		return nil, domain.Err(500, "Failed to submit application")
	}

	return &domain.CreatorApplication{
		ID:          appID,
		UserID:      user.UserID,
		Status:      "pending",
		Bio:         bio,
		SocialLinks: socialLinks,
		SubmittedAt: utils.FormatRFC3339UTC(now),
	}, nil
}

func (s *Services) GetCreatorDashboard(ctx context.Context, user *middleware.UserClaims) (*domain.CreatorDashboard, error) {
	var dashboard domain.CreatorDashboard

	_ = s.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM posts WHERE author_id = $1 AND deleted_at IS NULL`,
		user.UserID,
	).Scan(&dashboard.TotalPosts)

	_ = s.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(like_count), 0) FROM posts WHERE author_id = $1 AND deleted_at IS NULL`,
		user.UserID,
	).Scan(&dashboard.TotalLikes)

	return &dashboard, nil
}

func (s *Services) GetCreatorEarnings(ctx context.Context, user *middleware.UserClaims) (map[string]any, error) {
	return map[string]any{
		"earnings": []any{},
		"total":    0.0,
	}, nil
}

func (s *Services) GetCreatorCatalog(ctx context.Context, user *middleware.UserClaims, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id::text, title, description, price_cents, currency, visibility,
		        COALESCE(media_urls, '{}'), created_at, updated_at
		 FROM creator_catalog 
		 WHERE creator_id = $1::uuid AND deleted_at IS NULL
		 AND ($2::text = '' OR created_at < $2::timestamptz)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var items []any
	for rows.Next() {
		var item domain.CreatorCatalogItem
		var createdAt, updatedAt time.Time
		err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.PriceCents, &item.Currency, &item.Visibility,
			&item.Media, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		item.CreatedAt = utils.FormatRFC3339UTC(createdAt)
		item.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
		if item.Media == nil {
			item.Media = []string{}
		}
		items = append(items, item)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].(domain.CreatorCatalogItem).CreatedAt
	}

	return &domain.CursorPage{Data: items, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// PatchCreatorProfile atualiza a bio pública do criador em users e, se existir candidatura,
// mantém creator_applications.bio alinhado (mesma transação).
func (s *Services) PatchCreatorProfile(ctx context.Context, user *middleware.UserClaims, bio string) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return domain.Err(500, "Failed to update profile")
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE users SET bio = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		bio, user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to update profile")
	}
	if tag.RowsAffected() == 0 {
		return domain.Err(http.StatusNotFound, "User not found")
	}

	if _, err := tx.Exec(ctx,
		`UPDATE creator_applications SET bio = $1 WHERE user_id = $2`,
		bio, user.UserID,
	); err != nil {
		return domain.Err(500, "Failed to update profile")
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Err(500, "Failed to update profile")
	}
	if s.Logger != nil {
		s.Logger.Info("creator profile bio updated", "user_id", user.UserID)
	}
	return nil
}
