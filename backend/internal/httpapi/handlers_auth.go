package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresAt    time.Time  `json:"expires_at"`
	User         UserPublic `json:"user"`
}

type UserPublic struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Bio       string `json:"bio,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		respondValidationError(w, map[string]string{
			"fields": "email, username, and password are required",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	userID := uuid.New().String()
	now := time.Now().UTC()

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO users (id, email, username, password_hash, role, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		userID, req.Email, req.Username, string(hashedPassword), "user", now,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Unique constraint violation
			respondError(w, http.StatusConflict, "User already exists")
			return
		}
		// Other database error
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	accessToken, refreshToken, expiresAt, err := s.generateTokens(userID, req.Email, req.Username, []string{"user"})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Persist refresh token in sessions table
	refreshExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.db.Exec(r.Context(),
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), userID, refreshToken, refreshExpiry,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	respondCreated(w, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		User: UserPublic{
			ID:        userID,
			Username:  req.Username,
			Email:     req.Email,
			Role:      "user",
			CreatedAt: formatTimestamp(now),
		},
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req LoginRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var user struct {
		ID           string
		Username     string
		Email        string
		PasswordHash string
		Role         string
		CreatedAt    time.Time
	}

	err = s.db.QueryRow(r.Context(),
		`SELECT id, username, email, password_hash, role, created_at FROM users WHERE email = $1 AND deleted_at IS NULL`,
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	accessToken, refreshToken, expiresAt, err := s.generateTokens(user.ID, user.Email, user.Username, []string{user.Role})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Persist refresh token in sessions table
	refreshExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.db.Exec(r.Context(),
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), user.ID, refreshToken, refreshExpiry,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	respondJSON(w, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		User: UserPublic{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: formatTimestamp(user.CreatedAt),
		},
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "Refresh endpoint not yet implemented")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &req); err == nil && req.RefreshToken != "" {
		// Revoke the session in the database
		_, _ = s.db.Exec(r.Context(),
			`UPDATE sessions SET revoked_at = NOW() WHERE refresh_token = $1`,
			req.RefreshToken,
		)
	}

	respondNoContent(w)
}
