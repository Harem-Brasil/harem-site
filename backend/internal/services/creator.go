package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
)

func (s *Services) PostCreatorApply(ctx context.Context, user *middleware.UserClaims, bio string, socialLinks []string) (*domain.CreatorApplication, error) {
	appID := uuid.New().String()
	now := time.Now().UTC()

	_, err := s.DB.Exec(ctx,
		`INSERT INTO creator_applications (id, user_id, bio, social_links, status, submitted_at) 
		 VALUES ($1, $2, $3, $4, 'pending', $5)`,
		appID, user.UserID, bio, socialLinks, now,
	)
	if err != nil {
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
		`SELECT id, title, description, price_cents, currency, visibility, created_at 
		 FROM creator_catalog 
		 WHERE creator_id = $1 AND deleted_at IS NULL
		 AND ($2 = '' OR created_at < $2)
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
		var createdAt time.Time
		err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.PriceCents, &item.Currency, &item.Visibility, &createdAt)
		if err != nil {
			continue
		}
		item.CreatedAt = utils.FormatRFC3339UTC(createdAt)
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

func (s *Services) GetCreatorOrders(ctx context.Context, user *middleware.UserClaims, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id, buyer_id, item_id, status, amount_cents, currency, created_at 
		 FROM creator_orders 
		 WHERE creator_id = $1
		 AND ($2 = '' OR created_at < $2)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var orders []any
	for rows.Next() {
		var order domain.CreatorOrderRow
		var createdAt time.Time
		err := rows.Scan(&order.ID, &order.BuyerID, &order.ItemID, &order.Status, &order.AmountCents, &order.Currency, &createdAt)
		if err != nil {
			continue
		}
		order.CreatedAt = utils.FormatRFC3339UTC(createdAt)
		orders = append(orders, order)
	}

	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}

	nextCursor := ""
	if hasMore && len(orders) > 0 {
		nextCursor = orders[len(orders)-1].(domain.CreatorOrderRow).CreatedAt
	}

	return &domain.CursorPage{Data: orders, NextCursor: nextCursor, HasMore: hasMore}, nil
}
