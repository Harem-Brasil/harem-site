package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
)

func (s *Services) ListChatRooms(ctx context.Context, user *middleware.UserClaims) ([]any, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT cr.id, cr.name, cr.type, cr.description, cr.created_by, cr.created_at,
		        (SELECT COUNT(*) FROM chat_members WHERE room_id = cr.id) as member_count,
		        EXISTS(SELECT 1 FROM chat_members WHERE room_id = cr.id AND user_id = $1) as is_member
		 FROM chat_rooms cr
		 WHERE cr.deleted_at IS NULL
		 ORDER BY cr.created_at DESC`,
		user.UserID,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var rooms []any
	for rows.Next() {
		var room domain.ChatRoom
		err := rows.Scan(&room.ID, &room.Name, &room.Type, &room.Description, &room.CreatedBy,
			&room.CreatedAt, &room.MemberCount, &room.IsMember)
		if err != nil {
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

type CreateChatRoomBody struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

func (s *Services) CreateChatRoom(ctx context.Context, user *middleware.UserClaims, req CreateChatRoomBody) (*domain.ChatRoom, error) {
	if req.Name == "" {
		return nil, domain.ErrValidation("Name is required", map[string]string{"name": "Name is required"})
	}

	if req.Type == "" {
		req.Type = "public"
	}

	roomID := uuid.New().String()
	now := time.Now().UTC()

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, domain.Err(500, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO chat_rooms (id, name, type, description, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		roomID, req.Name, req.Type, req.Description, user.UserID, now,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create room")
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO chat_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'admin', $3)`,
		roomID, user.UserID, now,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to add creator as member")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.Err(500, "Failed to commit transaction")
	}

	return &domain.ChatRoom{
		ID:        roomID,
		Name:      req.Name,
		Type:      req.Type,
		CreatedBy: user.UserID,
		CreatedAt: utils.FormatRFC3339UTC(now),
		IsMember:  true,
	}, nil
}

func (s *Services) GetChatRoom(ctx context.Context, user *middleware.UserClaims, id string) (*domain.ChatRoom, error) {
	var room domain.ChatRoom
	err := s.DB.QueryRow(ctx,
		`SELECT cr.id, cr.name, cr.type, cr.description, cr.created_by, cr.created_at,
		        (SELECT COUNT(*) FROM chat_members WHERE room_id = cr.id) as member_count,
		        EXISTS(SELECT 1 FROM chat_members WHERE room_id = cr.id AND user_id = $2) as is_member
		 FROM chat_rooms cr
		 WHERE cr.id = $1 AND cr.deleted_at IS NULL`,
		id, user.UserID,
	).Scan(&room.ID, &room.Name, &room.Type, &room.Description, &room.CreatedBy,
		&room.CreatedAt, &room.MemberCount, &room.IsMember)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "Room not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	return &room, nil
}

func (s *Services) JoinChatRoom(ctx context.Context, user *middleware.UserClaims, roomID string) error {
	_, err := s.DB.Exec(ctx,
		`INSERT INTO chat_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'member', NOW()) ON CONFLICT DO NOTHING`,
		roomID, user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to join room")
	}
	return nil
}

func (s *Services) ListChatMessages(ctx context.Context, roomID, cursor string) (*domain.CursorPage, error) {
	limit := 50

	rows, err := s.DB.Query(ctx,
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
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var messages []any
	for rows.Next() {
		var msg domain.ChatMessage
		var sender domain.UserPublic
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
		nextCursor = messages[len(messages)-1].(domain.ChatMessage).CreatedAt
	}

	return &domain.CursorPage{Data: messages, NextCursor: nextCursor, HasMore: hasMore}, nil
}
