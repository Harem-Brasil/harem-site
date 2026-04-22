package services

import (
	"context"
	"time"

	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/utils"
)

type AdminStats struct {
	TotalUsers          int `json:"total_users"`
	ActiveUsers         int `json:"active_users"`
	TotalPosts          int `json:"total_posts"`
	TotalCreators       int `json:"total_creators"`
	ActiveSubscriptions int `json:"active_subscriptions"`
}

func (s *Services) AdminListUsers(ctx context.Context, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id, username, email, role, created_at FROM users
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
		var createdAt time.Time
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &createdAt)
		if err != nil {
			continue
		}
		u.CreatedAt = utils.FormatRFC3339UTC(createdAt)
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

type AdminUpdateRoleBody struct {
	Role string `json:"role"`
}

func (s *Services) AdminUpdateRole(ctx context.Context, id string, req AdminUpdateRoleBody) error {
	validRoles := map[string]bool{"guest": true, "user": true, "creator": true, "moderator": true, "admin": true}
	if !validRoles[req.Role] {
		return domain.ErrValidation("Invalid role", map[string]string{"role": "Invalid role"})
	}

	_, err := s.DB.Exec(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`,
		req.Role, id,
	)
	if err != nil {
		return domain.Err(500, "Failed to update role")
	}
	return nil
}

func (s *Services) AdminDeleteUser(ctx context.Context, id string) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE users SET deleted_at = NOW(), email = CONCAT('deleted_', id, '@example.com') WHERE id = $1`,
		id,
	)
	if err != nil {
		return domain.Err(500, "Failed to delete user")
	}
	return nil
}

func (s *Services) AdminStats(ctx context.Context) (*AdminStats, error) {
	var stats AdminStats

	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&stats.TotalUsers); err != nil {
		s.Logger.Error("failed to get total users", "error", err)
	}
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND last_seen_at > NOW() - INTERVAL '30 days'`).Scan(&stats.ActiveUsers); err != nil {
		s.Logger.Error("failed to get active users", "error", err)
	}
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL`).Scan(&stats.TotalPosts); err != nil {
		s.Logger.Error("failed to get total posts", "error", err)
	}
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'creator'`).Scan(&stats.TotalCreators); err != nil {
		s.Logger.Error("failed to get total creators", "error", err)
	}
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM subscriptions WHERE status = 'active'`).Scan(&stats.ActiveSubscriptions); err != nil {
		s.Logger.Error("failed to get active subscriptions", "error", err)
	}

	return &stats, nil
}

func (s *Services) AdminAuditLog(ctx context.Context, cursor string) (*domain.CursorPage, error) {
	limit := 50

	rows, err := s.DB.Query(ctx,
		`SELECT id, user_id, action, resource, details, ip_address, created_at FROM audit_log
		 WHERE ($1 = '' OR created_at < $1)
		 ORDER BY created_at DESC LIMIT $2`,
		cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var entries []any
	for rows.Next() {
		var e domain.AuditLogEntry
		var createdAt time.Time
		err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Resource, &e.Details, &e.IP, &createdAt)
		if err != nil {
			continue
		}
		e.CreatedAt = utils.FormatRFC3339UTC(createdAt)
		entries = append(entries, e)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	nextCursor := ""
	if hasMore && len(entries) > 0 {
		nextCursor = entries[len(entries)-1].(domain.AuditLogEntry).CreatedAt
	}

	return &domain.CursorPage{Data: entries, NextCursor: nextCursor, HasMore: hasMore}, nil
}
