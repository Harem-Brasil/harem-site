package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

type ChatRoom struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	CreatedBy   string `json:"created_by"`
	MemberCount int    `json:"member_count"`
	CreatedAt   string `json:"created_at"`
	IsMember    bool   `json:"is_member"`
}

type ChatMessage struct {
	ID        string     `json:"id"`
	RoomID    string     `json:"room_id"`
	SenderID  string     `json:"sender_id"`
	Content   string     `json:"content"`
	CreatedAt string     `json:"created_at"`
	Sender    UserPublic `json:"sender,omitempty"`
}

func (s *Server) handleListChatRooms(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())

	rows, err := s.db.Query(r.Context(),
		`SELECT cr.id, cr.name, cr.type, cr.description, cr.created_by, cr.created_at,
		        (SELECT COUNT(*) FROM chat_members WHERE room_id = cr.id) as member_count,
		        EXISTS(SELECT 1 FROM chat_members WHERE room_id = cr.id AND user_id = $1) as is_member
		 FROM chat_rooms cr
		 WHERE cr.deleted_at IS NULL
		 ORDER BY cr.created_at DESC`,
		user.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var rooms []any
	for rows.Next() {
		var room ChatRoom
		err := rows.Scan(&room.ID, &room.Name, &room.Type, &room.Description, &room.CreatedBy,
			&room.CreatedAt, &room.MemberCount, &room.IsMember)
		if err != nil {
			continue
		}
		rooms = append(rooms, room)
	}

	respondJSON(w, rooms)
}

func (s *Server) handleCreateChatRoom(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Name == "" {
		respondValidationError(w, map[string]string{"name": "Name is required"})
		return
	}

	if req.Type == "" {
		req.Type = "public"
	}

	roomID := uuid.New().String()
	now := time.Now().UTC()

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),
		`INSERT INTO chat_rooms (id, name, type, description, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		roomID, req.Name, req.Type, req.Description, user.UserID, now,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create room")
		return
	}

	_, err = tx.Exec(r.Context(),
		`INSERT INTO chat_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'admin', $3)`,
		roomID, user.UserID, now,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add creator as member")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondCreated(w, ChatRoom{
		ID:        roomID,
		Name:      req.Name,
		Type:      req.Type,
		CreatedBy: user.UserID,
		CreatedAt: formatTimestamp(now),
		IsMember:  true,
	})
}

func (s *Server) handleGetChatRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := httpmw.GetUser(r.Context())

	var room ChatRoom
	err := s.db.QueryRow(r.Context(),
		`SELECT cr.id, cr.name, cr.type, cr.description, cr.created_by, cr.created_at,
		        (SELECT COUNT(*) FROM chat_members WHERE room_id = cr.id) as member_count,
		        EXISTS(SELECT 1 FROM chat_members WHERE room_id = cr.id AND user_id = $2) as is_member
		 FROM chat_rooms cr
		 WHERE cr.id = $1 AND cr.deleted_at IS NULL`,
		id, user.UserID,
	).Scan(&room.ID, &room.Name, &room.Type, &room.Description, &room.CreatedBy,
		&room.CreatedAt, &room.MemberCount, &room.IsMember)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "Room not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, room)
}

func (s *Server) handleJoinChatRoom(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	roomID := chi.URLParam(r, "id")

	_, err := s.db.Exec(r.Context(),
		`INSERT INTO chat_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'member', NOW()) ON CONFLICT DO NOTHING`,
		roomID, user.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to join room")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	cursor := r.URL.Query().Get("cursor")
	limit := 50

	rows, err := s.db.Query(r.Context(),
		`SELECT cm.id, cm.room_id, cm.sender_id, cm.content, cm.created_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 WHERE cm.room_id = $1 AND cm.deleted_at IS NULL
		 AND ($2 = '' OR cm.created_at < $2)
		 ORDER BY cm.created_at DESC LIMIT $3`,
		roomID, cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var messages []any
	for rows.Next() {
		var msg ChatMessage
		var sender UserPublic
		err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderID, &msg.Content, &msg.CreatedAt,
			&sender.ID, &sender.Username, &sender.Role, &sender.AvatarURL)
		if err != nil {
			continue
		}
		msg.Sender = sender
		messages = append(messages, msg)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	nextCursor := ""
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].(ChatMessage).CreatedAt
	}

	respondJSON(w, CursorPage{
		Data:       messages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
