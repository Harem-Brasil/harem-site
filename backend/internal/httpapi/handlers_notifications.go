package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data,omitempty"`
	ReadAt    *string        `json:"read_at,omitempty"`
	CreatedAt string         `json:"created_at"`
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	unreadOnly := r.URL.Query().Get("unread") == "true"
	limit := 20

	query := `SELECT id, user_id, type, title, body, data, read_at, created_at FROM notifications 
	          WHERE user_id = $1 AND ($3 = false OR read_at IS NULL)
	          AND ($2 = '' OR created_at < $2)
	          ORDER BY created_at DESC LIMIT $4`

	rows, err := s.db.Query(r.Context(), query, user.UserID, cursor, unreadOnly, limit+1)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var notifications []any
	for rows.Next() {
		var n Notification
		var readAt *time.Time
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &readAt, &n.CreatedAt)
		if err != nil {
			continue
		}
		if readAt != nil {
			formatted := formatTimestamp(*readAt)
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
		nextCursor = notifications[len(notifications)-1].(Notification).CreatedAt
	}

	respondJSON(w, CursorPage{
		Data:       notifications,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	_, err := s.db.Exec(r.Context(),
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2`,
		id, user.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var count int
	err := s.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		user.UserID,
	).Scan(&count)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, map[string]int{"unread_count": count})
}

func (s *Server) createNotification(userID, notifType, title, body string, data map[string]any) error {
	id := uuid.New().String()
	_, err := s.db.Exec(context.Background(),
		`INSERT INTO notifications (id, user_id, type, title, body, data, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		id, userID, notifType, title, body, data,
	)
	return err
}
