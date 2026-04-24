package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/utils"
)

func (s *Services) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	fieldErrors, ok := req.Validate()
	if !ok {
		return nil, domain.ErrValidation("One or more fields failed validation", fieldErrors)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.Err(500, "Failed to process password")
	}

	userID := uuid.New().String()
	now := time.Now().UTC()

	_, err = s.DB.Exec(ctx,
		`INSERT INTO users (id, email, username, password_hash, role, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		userID, req.Email, req.Username, string(hashedPassword), "user", now,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.Err(409, "User already exists")
		}
		return nil, domain.Err(500, "Database error")
	}

	accessToken, refreshToken, expiresAt, err := s.generateTokens(userID, req.Email, req.Username, []string{"user"})
	if err != nil {
		return nil, domain.Err(500, "Failed to generate tokens")
	}

	refreshExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.DB.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), userID, refreshToken, refreshExpiry,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create session")
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		User: domain.UserPublic{
			ID:        userID,
			Username:  req.Username,
			Email:     req.Email,
			Role:      "user",
			CreatedAt: utils.FormatRFC3339UTC(now),
		},
	}, nil
}

func (s *Services) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	fieldErrors, ok := req.Validate()
	if !ok {
		return nil, domain.ErrValidation("One or more fields failed validation", fieldErrors)
	}

	var user struct {
		ID           string
		Username     string
		Email        string
		PasswordHash string
		Role         string
		CreatedAt    time.Time
	}

	err := s.DB.QueryRow(ctx,
		`SELECT id, username, email, password_hash, role, created_at FROM users WHERE email = $1 AND deleted_at IS NULL`,
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(401, "Invalid credentials")
		}
		return nil, domain.Err(500, "Database error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.Err(401, "Invalid credentials")
	}

	accessToken, refreshToken, expiresAt, err := s.generateTokens(user.ID, user.Email, user.Username, []string{user.Role})
	if err != nil {
		return nil, domain.Err(500, "Failed to generate tokens")
	}

	refreshExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.DB.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), user.ID, refreshToken, refreshExpiry,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create session")
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		User: domain.UserPublic{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: utils.FormatRFC3339UTC(user.CreatedAt),
		},
	}, nil
}

type RefreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Services) Refresh(ctx context.Context, req RefreshBody) (map[string]any, error) {
	if req.RefreshToken == "" {
		return nil, domain.ErrValidation("refresh_token required", map[string]string{"refresh_token": "Required"})
	}

	var session struct {
		UserID    string
		ExpiresAt time.Time
	}
	err := s.DB.QueryRow(ctx,
		`SELECT user_id, expires_at FROM sessions WHERE refresh_token = $1 AND revoked_at IS NULL`,
		req.RefreshToken,
	).Scan(&session.UserID, &session.ExpiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(401, "Invalid refresh token")
		}
		return nil, domain.Err(500, "Database error")
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, domain.Err(401, "Refresh token expired")
	}

	var user struct {
		ID       string
		Email    string
		Username string
		Role     string
	}
	err = s.DB.QueryRow(ctx,
		`SELECT id, email, username, role FROM users WHERE id = $1 AND deleted_at IS NULL`,
		session.UserID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.Role)

	if err != nil {
		return nil, domain.Err(500, "User not found")
	}

	accessToken, refreshToken, expiresAt, err := s.generateTokens(user.ID, user.Email, user.Username, []string{user.Role})
	if err != nil {
		return nil, domain.Err(500, "Failed to generate tokens")
	}

	if _, err := s.DB.Exec(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE refresh_token = $1`,
		req.RefreshToken,
	); err != nil {
		tokenHint := req.RefreshToken
		if len(tokenHint) > 8 {
			tokenHint = tokenHint[:8] + "..."
		}
		s.Logger.Error("failed to revoke old refresh token", "error", err, "refresh_token", tokenHint)
	}

	refreshExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.DB.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), user.ID, refreshToken, refreshExpiry,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create session")
	}

	return map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_at":    expiresAt,
	}, nil
}

type LogoutBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Services) Logout(ctx context.Context, req LogoutBody) error {
	if req.RefreshToken != "" {
		_, _ = s.DB.Exec(ctx,
			`UPDATE sessions SET revoked_at = NOW() WHERE refresh_token = $1`,
			req.RefreshToken,
		)
	}
	return nil
}

func (s *Services) LogoutAll(ctx context.Context) error {
	return domain.Err(501, "Logout all sessions not yet implemented")
}

func (s *Services) OAuthAuthorize(ctx context.Context, provider string) error {
	return domain.Err(501, "OAuth authorization not yet implemented")
}

func (s *Services) OAuthCallback(ctx context.Context, provider string) error {
	return domain.Err(501, "OAuth callback not yet implemented")
}

func (s *Services) EmailVerify(ctx context.Context) error {
	return domain.Err(501, "Email verification not yet implemented")
}

func (s *Services) PasswordForgot(ctx context.Context) error {
	return domain.Err(501, "Password forgot not yet implemented")
}

func (s *Services) PasswordReset(ctx context.Context) error {
	return domain.Err(501, "Password reset not yet implemented")
}
