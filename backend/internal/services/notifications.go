package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
)

func (s *Services) ListNotifications(ctx context.Context, user *middleware.UserClaims, cursor string, unreadOnly bool) (*domain.CursorPage, error) {
	limit := 20

	query := `SELECT id, user_id, type, title, body, data, read_at, created_at FROM notifications 
	          WHERE user_id = $1 AND ($3 = false OR read_at IS NULL)
	          AND ($2 = '' OR created_at < $2)
	          ORDER BY created_at DESC LIMIT $4`

	rows, err := s.DB.Query(ctx, query, user.UserID, cursor, unreadOnly, limit+1)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var notifications []any
	for rows.Next() {
		var n domain.Notification
		var readAt *time.Time
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &readAt, &n.CreatedAt)
		if err != nil {
			continue
		}
		if readAt != nil {
			formatted := utils.FormatRFC3339UTC(*readAt)
			n.ReadAt = &formatted
		}
		notifications = append(notifications, n)
	}

	hasMore := len(notifications) > limit
	if hasMore {
		notifications = notifications[:limit]
	}

	nextCursor := ""
	if hasMore && len(notifications) > 0 {
		nextCursor = notifications[len(notifications)-1].(domain.Notification).CreatedAt
	}

	return &domain.CursorPage{Data: notifications, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Services) MarkNotificationRead(ctx context.Context, user *middleware.UserClaims, id string) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2`,
		id, user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to mark notification as read")
	}
	return nil
}

func (s *Services) UnreadNotificationCount(ctx context.Context, user *middleware.UserClaims) (int, error) {
	var count int
	err := s.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		user.UserID,
	).Scan(&count)

	if err != nil {
		return 0, domain.Err(500, "Database error")
	}
	return count, nil
}

func (s *Services) createNotification(ctx context.Context, userID, notifType, title, body string, data map[string]any) error {
	id := uuid.New().String()
	_, err := s.DB.Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, title, body, data, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		id, userID, notifType, title, body, data,
	)
	return err
}
