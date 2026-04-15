package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var u struct {
		ID        string
		Username  string
		Email     string
		Role      string
		Bio       pgtype.Text
		AvatarURL pgtype.Text
		CreatedAt pgtype.Timestamptz
	}

	err := s.db.QueryRow(r.Context(),
		`SELECT id, username, email, role, bio, avatar_url, created_at FROM users WHERE id = $1`,
		user.UserID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Bio, &u.AvatarURL, &u.CreatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, UserPublic{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Bio:       u.Bio.String,
		AvatarURL: u.AvatarURL.String,
		CreatedAt: formatTimestamp(u.CreatedAt.Time),
	})
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
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

	var updates map[string]any
	if err := json.Unmarshal(body, &updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	_, err = s.db.Exec(r.Context(),
		`UPDATE users SET username = COALESCE($1, username), bio = COALESCE($2, bio), updated_at = NOW() WHERE id = $3`,
		updates["username"], updates["bio"], user.UserID,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	_, err := s.db.Exec(r.Context(),
		`UPDATE users SET deleted_at = NOW(), email = CONCAT('deleted_', id, '@example.com') WHERE id = $1`,
		user.UserID,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var u struct {
		ID        string
		Username  string
		Role      string
		Bio       pgtype.Text
		AvatarURL pgtype.Text
		CreatedAt pgtype.Timestamptz
	}

	err := s.db.QueryRow(r.Context(),
		`SELECT id, username, role, bio, avatar_url, created_at FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Bio, &u.AvatarURL, &u.CreatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, UserPublic{
		ID:        u.ID,
		Username:  u.Username,
		Role:      u.Role,
		Bio:       u.Bio.String,
		AvatarURL: u.AvatarURL.String,
		CreatedAt: formatTimestamp(u.CreatedAt.Time),
	})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT id, username, role, avatar_url, created_at FROM users 
		 WHERE deleted_at IS NULL AND ($1 = '' OR id > $1) 
		 ORDER BY id LIMIT $2`,
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
		nextCursor = users[len(users)-1].(UserPublic).ID
	}

	respondJSON(w, CursorPage{
		Data:       users,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
