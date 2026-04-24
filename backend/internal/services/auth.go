package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
)

const (
	refreshTokenExpiry = 7 * 24 * time.Hour
	accessTokenExpiry  = 15 * time.Minute
	bcryptCost         = 12
)

// execer abstracts pgxpool.Pool and pgx.Tx so storeRefreshToken works with either.
type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// SessionMeta carries HTTP request metadata for refresh token auditing.
type SessionMeta struct {
	IP        string
	UserAgent string
}

// storeRefreshToken inserts a row into refresh_tokens using a pre-hashed tokenHash.
func (s *Services) storeRefreshToken(ctx context.Context, exec execer, userID, tokenID, tokenHash string, meta *SessionMeta) error {
	refreshExpiry := time.Now().UTC().Add(refreshTokenExpiry)
	_, err := exec.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_id, token_hash, expires_at, last_used_at, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.New().String(), userID, tokenID, tokenHash, refreshExpiry, time.Now().UTC(), meta.IP, meta.UserAgent,
	)
	return err
}

func (s *Services) Register(ctx context.Context, req domain.RegisterRequest, meta *SessionMeta) (*domain.AuthResponse, error) {
	fieldErrors, ok := req.Validate()
	if !ok {
		return nil, domain.ErrValidation("One or more fields failed validation", fieldErrors)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, domain.Err(500, "Failed to process password")
	}

	userID := uuid.New().String()
	now := time.Now().UTC()

	_, err = s.DB.Exec(ctx,
		`INSERT INTO users (id, email, screen_name, password_hash, role, accept_terms_version, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		userID, req.Email, req.ScreenName, string(hashedPassword), "user", req.AcceptTermsVersion, now,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.Err(409, "User already exists")
		}
		return nil, domain.Err(500, "Database error")
	}

	accessToken, refreshToken, tokenID, expiresAt, err := s.generateTokens(userID, req.Email, req.ScreenName, []string{"user"})
	if err != nil {
		return nil, domain.Err(500, "Failed to generate tokens")
	}

	_, secret, ok := splitRefreshToken(refreshToken)
	if !ok {
		return nil, domain.Err(500, "Failed to process refresh token")
	}

	tokenHash := sha256Hash(secret)

	if err := s.storeRefreshToken(ctx, s.DB, userID, tokenID, tokenHash, meta); err != nil {
		return nil, domain.Err(500, "Failed to create session")
	}

	if s.Logger != nil {
		s.Logger.Info("auth register success",
			"user_id", userID,
			"role", "user",
		)
	}

	return &domain.AuthResponse{
		AccessToken:      accessToken,
		AccessExpiresIn:  int64(accessTokenExpiry.Seconds()),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int64(refreshTokenExpiry.Seconds()),
		TokenType:        "Bearer",
		ExpiresAt:        expiresAt,
		User: domain.UserPublic{
			ID:         userID,
			ScreenName: req.ScreenName,
			Email:      req.Email,
			Role:       "user",
			CreatedAt:  utils.FormatRFC3339UTC(now),
		},
	}, nil
}

func (s *Services) Login(ctx context.Context, req domain.LoginRequest, meta *SessionMeta) (*domain.AuthResponse, error) {
	fieldErrors, ok := req.Validate()
	if !ok {
		return nil, domain.ErrValidation("One or more fields failed validation", fieldErrors)
	}

	var user struct {
		ID           string
		ScreenName   string
		Email        string
		PasswordHash string
		Role         string
		CreatedAt    time.Time
	}

	err := s.DB.QueryRow(ctx,
		`SELECT id, screen_name, email, password_hash, role, created_at FROM users WHERE email = $1 AND deleted_at IS NULL`,
		req.Email,
	).Scan(&user.ID, &user.ScreenName, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(401, "Invalid credentials")
		}
		return nil, domain.Err(500, "Database error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		if s.Logger != nil {
			s.Logger.Warn("auth login failure",
				"reason", "invalid_credentials",
				"user_id", user.ID,
			)
		}
		return nil, domain.Err(401, "Invalid credentials")
	}

	accessToken, refreshToken, tokenID, expiresAt, err := s.generateTokens(user.ID, user.Email, user.ScreenName, []string{user.Role})
	if err != nil {
		return nil, domain.Err(500, "Failed to generate tokens")
	}

	_, secret, ok := splitRefreshToken(refreshToken)
	if !ok {
		return nil, domain.Err(500, "Failed to process refresh token")
	}

	tokenHash := sha256Hash(secret)

	if err := s.storeRefreshToken(ctx, s.DB, user.ID, tokenID, tokenHash, meta); err != nil {
		return nil, domain.Err(500, "Failed to create session")
	}

	if s.Logger != nil {
		s.Logger.Info("auth login success",
			"user_id", user.ID,
			"role", user.Role,
		)
	}

	return &domain.AuthResponse{
		AccessToken:      accessToken,
		AccessExpiresIn:  int64(accessTokenExpiry.Seconds()),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int64(refreshTokenExpiry.Seconds()),
		TokenType:        "Bearer",
		ExpiresAt:        expiresAt,
		User: domain.UserPublic{
			ID:         user.ID,
			ScreenName: user.ScreenName,
			Email:      user.Email,
			Role:       user.Role,
			CreatedAt:  utils.FormatRFC3339UTC(user.CreatedAt),
		},
	}, nil
}

type RefreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Services) Refresh(ctx context.Context, req RefreshBody, meta *SessionMeta) (*domain.AuthResponse, error) {
	if req.RefreshToken == "" {
		return nil, domain.ErrValidation("refresh_token required", map[string]string{"refresh_token": "Required"})
	}

	tokenID, secret, ok := splitRefreshToken(req.RefreshToken)
	if !ok {
		return nil, domain.Err(401, "Invalid refresh token format")
	}
	if _, err := uuid.Parse(tokenID); err != nil {
		return nil, domain.Err(401, "Invalid refresh token format")
	}

	var session struct {
		ID        string
		UserID    string
		TokenHash string
		ExpiresAt time.Time
		RevokedAt *time.Time
	}
	err := s.DB.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at FROM refresh_tokens WHERE token_id = $1`,
		tokenID,
	).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.RevokedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(401, "Invalid refresh token")
		}
		return nil, domain.Err(500, "Database error")
	}

	if session.RevokedAt != nil {
		return nil, domain.Err(401, "Refresh token revoked")
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, domain.Err(401, "Refresh token expired")
	}

	if sha256Hash(secret) != session.TokenHash {
		return nil, domain.Err(401, "Invalid refresh token")
	}

	var user struct {
		ID         string
		Email      string
		ScreenName string
		Role       string
	}
	err = s.DB.QueryRow(ctx,
		`SELECT id, email, screen_name, role FROM users WHERE id = $1 AND deleted_at IS NULL`,
		session.UserID,
	).Scan(&user.ID, &user.Email, &user.ScreenName, &user.Role)

	if err != nil {
		return nil, domain.Err(500, "User not found")
	}

	accessToken, refreshToken, newTokenID, expiresAt, err := s.generateTokens(user.ID, user.Email, user.ScreenName, []string{user.Role})
	if err != nil {
		return nil, domain.Err(500, "Failed to generate tokens")
	}

	_, newSecret, ok := splitRefreshToken(refreshToken)
	if !ok {
		return nil, domain.Err(500, "Failed to process refresh token")
	}

	// Compute SHA-256 hash BEFORE opening transaction to avoid holding DB locks.
	newTokenHash := sha256Hash(newSecret)

	// Atomic rotation: revoke old token + insert new token in a single transaction.
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer tx.Rollback(ctx)

	res, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET last_used_at = NOW(), revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`,
		session.ID,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to revoke old refresh token")
	}

	if res.RowsAffected() == 0 {
		return nil, domain.Err(401, "Refresh token already used or revoked")
	}

	if err := s.storeRefreshToken(ctx, tx, user.ID, newTokenID, string(newTokenHash), meta); err != nil {
		return nil, domain.Err(500, "Failed to create session")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.Err(500, "Failed to commit transaction")
	}

	return &domain.AuthResponse{
		AccessToken:      accessToken,
		AccessExpiresIn:  int64(accessTokenExpiry.Seconds()),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int64(refreshTokenExpiry.Seconds()),
		TokenType:        "Bearer",
		ExpiresAt:        expiresAt,
		User: domain.UserPublic{
			ID:         user.ID,
			Email:      user.Email,
			ScreenName: user.ScreenName,
			Role:       user.Role,
		},
	}, nil
}

type LogoutBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Services) Logout(ctx context.Context, req LogoutBody) error {
	if req.RefreshToken != "" {
		tokenID, _, ok := splitRefreshToken(req.RefreshToken)
		if ok {
			if _, err := s.DB.Exec(ctx,
				`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_id = $1 AND revoked_at IS NULL`,
				tokenID,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Services) LogoutAll(ctx context.Context, user *middleware.UserClaims) error {
	if user == nil {
		return domain.Err(401, "Unauthorized")
	}
	_, err := s.DB.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
		user.UserID,
	)
	return err
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

// sha256Hash returns the hex-encoded SHA-256 hash of the input.
// Used for refresh token secrets which are already high-entropy crypto-random values,
// so bcrypt is unnecessary overhead — SHA-256 provides equivalent security.
func sha256Hash(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
