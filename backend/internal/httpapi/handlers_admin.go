package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT id, username, email, role, created_at FROM users
		 WHERE deleted_at IS NULL AND ($1 = '' OR created_at < $1)
		 ORDER BY created_at DESC LIMIT $2`,
		cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var users []any
	for rows.Next() {
		var u UserPublic
		var createdAt time.Time
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &createdAt)
		if err != nil {
			continue
		}
		u.CreatedAt = formatTimestamp(createdAt)
		users = append(users, u)
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}

	nextCursor := ""
	if hasMore && len(users) > 0 {
		nextCursor = users[len(users)-1].(UserPublic).CreatedAt
	}

	respondJSON(w, CursorPage{
		Data:       users,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (s *Server) handleAdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	validRoles := map[string]bool{"guest": true, "user": true, "creator": true, "moderator": true, "admin": true}
	if !validRoles[req.Role] {
		respondValidationError(w, map[string]string{"role": "Invalid role"})
		return
	}

	_, err = s.db.Exec(r.Context(),
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`,
		req.Role, id,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := s.db.Exec(r.Context(),
		`UPDATE users SET deleted_at = NOW(), email = CONCAT('deleted_', id, '@example.com') WHERE id = $1`,
		id,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalUsers          int `json:"total_users"`
		ActiveUsers         int `json:"active_users"`
		TotalPosts          int `json:"total_posts"`
		TotalCreators       int `json:"total_creators"`
		ActiveSubscriptions int `json:"active_subscriptions"`
	}

	if err := s.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&stats.TotalUsers); err != nil {
		slog.Error("failed to get total users", "error", err)
	}
	if err := s.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND last_seen_at > NOW() - INTERVAL '30 days'`).Scan(&stats.ActiveUsers); err != nil {
		slog.Error("failed to get active users", "error", err)
	}
	if err := s.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL`).Scan(&stats.TotalPosts); err != nil {
		slog.Error("failed to get total posts", "error", err)
	}
	if err := s.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM users WHERE role = 'creator'`).Scan(&stats.TotalCreators); err != nil {
		slog.Error("failed to get total creators", "error", err)
	}
	if err := s.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM subscriptions WHERE status = 'active'`).Scan(&stats.ActiveSubscriptions); err != nil {
		slog.Error("failed to get active subscriptions", "error", err)
	}

	respondJSON(w, stats)
}

type AuditLogEntry struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Details   map[string]any `json:"details,omitempty"`
	IP        string         `json:"ip,omitempty"`
	CreatedAt string         `json:"created_at"`
}

func (s *Server) handleAdminAuditLog(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 50

	rows, err := s.db.Query(r.Context(),
		`SELECT id, user_id, action, resource, details, ip_address, created_at FROM audit_log
		 WHERE ($1 = '' OR created_at < $1)
		 ORDER BY created_at DESC LIMIT $2`,
		cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var entries []any
	for rows.Next() {
		var e AuditLogEntry
		var createdAt time.Time
		err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Resource, &e.Details, &e.IP, &createdAt)
		if err != nil {
			continue
		}
		e.CreatedAt = formatTimestamp(createdAt)
		entries = append(entries, e)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	nextCursor := ""
	if hasMore && len(entries) > 0 {
		nextCursor = entries[len(entries)-1].(AuditLogEntry).CreatedAt
	}

	respondJSON(w, CursorPage{
		Data:       entries,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
