package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/datasources"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		req        domain.RegisterRequest
		wantFields []string
	}{
		{
			name:       "valid request",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: nil,
		},
		{
			name:       "weak password - too short",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "Ab1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no uppercase",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "securepass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no special char",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "SecurePass1", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "missing email",
			req:        domain.RegisterRequest{ScreenName: "testuser", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			req:        domain.RegisterRequest{Email: "not-an-email", ScreenName: "testuser", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"email"},
		},
		{
			name:       "missing screen_name",
			req:        domain.RegisterRequest{Email: "test@example.com", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"screen_name"},
		},
		{
			name:       "screen_name too short",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "a", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"screen_name"},
		},
		{
			name:       "missing password",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "missing accept_terms_version",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "SecurePass1!"},
			wantFields: []string{"accept_terms_version"},
		},
		{
			name:       "all fields missing",
			req:        domain.RegisterRequest{},
			wantFields: []string{"email", "screen_name", "password", "accept_terms_version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, ok := tt.req.Validate()
			for _, field := range tt.wantFields {
				if _, exists := errors[field]; !exists {
					t.Errorf("expected validation error for field %q, got none", field)
				}
			}
			if tt.wantFields == nil && !ok {
				t.Errorf("expected no validation errors, got %v", errors)
			}
			if tt.wantFields != nil && ok {
				t.Errorf("expected validation errors, got none")
			}
		})
	}
}

func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		req        domain.LoginRequest
		wantFields []string
	}{
		{
			name:       "valid request",
			req:        domain.LoginRequest{Email: "test@example.com", Password: "anything"},
			wantFields: nil,
		},
		{
			name:       "missing email",
			req:        domain.LoginRequest{Password: "anything"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			req:        domain.LoginRequest{Email: "not-an-email", Password: "anything"},
			wantFields: []string{"email"},
		},
		{
			name:       "missing password",
			req:        domain.LoginRequest{Email: "test@example.com"},
			wantFields: []string{"password"},
		},
		{
			name:       "all fields missing",
			req:        domain.LoginRequest{},
			wantFields: []string{"email", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, ok := tt.req.Validate()
			for _, field := range tt.wantFields {
				if _, exists := errors[field]; !exists {
					t.Errorf("expected validation error for field %q, got none", field)
				}
			}
			if tt.wantFields == nil && !ok {
				t.Errorf("expected no validation errors, got %v", errors)
			}
			if tt.wantFields != nil && ok {
				t.Errorf("expected validation errors, got none")
			}
		})
	}
}

func TestSha256Hash(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want64 bool
	}{
		{"empty string", "", true},
		{"short input", "hello", true},
		{"long input", strings.Repeat("a", 1000), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sha256Hash(tt.input)
			if len(got) != 64 {
				t.Errorf("sha256Hash(%q) = %d chars, want 64", tt.input, len(got))
			}
			if sha256Hash(tt.input) != got {
				t.Errorf("sha256Hash is not deterministic for %q", tt.input)
			}
			if tt.input != "" && sha256Hash(tt.input+"x") == got {
				t.Errorf("sha256Hash collision for %q", tt.input)
			}
		})
	}
}

func TestSplitRefreshToken(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantSecret string
		wantOK     bool
	}{
		{"valid", "abc123.secretpart", "abc123", "secretpart", true},
		{"no dot", "abc123secretpart", "", "", false},
		{"empty", "", "", "", false},
		{"dot only", ".", "", "", true},
		{"multiple dots", "abc123.secret.extra", "abc123", "secret.extra", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, secret, ok := splitRefreshToken(tt.input)
			if ok != tt.wantOK {
				t.Errorf("splitRefreshToken(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok {
				if id != tt.wantID {
					t.Errorf("splitRefreshToken(%q) id = %q, want %q", tt.input, id, tt.wantID)
				}
				if secret != tt.wantSecret {
					t.Errorf("splitRefreshToken(%q) secret = %q, want %q", tt.input, secret, tt.wantSecret)
				}
			}
		})
	}
}

func TestLogoutOwnershipCheck(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		tokenOwnerID string
		shouldRevoke bool
	}{
		{"same user", "user-1", "user-1", true},
		{"different user", "user-1", "user-2", false},
		{"empty owner", "user-1", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRevoke := tt.userID == tt.tokenOwnerID && tt.tokenOwnerID != ""
			if shouldRevoke != tt.shouldRevoke {
				t.Errorf("ownership check: userID=%q tokenOwnerID=%q shouldRevoke=%v want=%v",
					tt.userID, tt.tokenOwnerID, shouldRevoke, tt.shouldRevoke)
			}
		})
	}
}

func TestLogoutAllRequiresAuth(t *testing.T) {
	svc := Services{}
	err := svc.LogoutAll(nil, nil)
	if err == nil {
		t.Error("expected error for nil UserClaims, got nil")
	}
}

func TestLogoutIdempotency(t *testing.T) {
	svc := Services{Dependencies: Dependencies{Logger: nil}}
	user := &middleware.UserClaims{UserID: "user-1"}

	err := svc.Logout(nil, user, LogoutBody{RefreshToken: ""})
	if err != nil {
		t.Errorf("Logout with empty token should be no-op, got: %v", err)
	}

	err = svc.Logout(nil, user, LogoutBody{RefreshToken: "not-a-valid-token"})
	if err != nil {
		t.Errorf("Logout with malformed token should be no-op, got: %v", err)
	}
}

// --- Integration tests: refresh rotation & revocation (§3, §6.2) ---

func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://harem:harem@localhost:5432/harem?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := datasources.NewPostgresPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("Test database unavailable: %v", err)
	}
	return pool
}

func mustHash(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

func createTestUser(ctx context.Context, t *testing.T, db *pgxpool.Pool) (userID, email, screenName string) {
	t.Helper()
	userID = uuid.New().String()
	email = userID + "@test.local"
	screenName = "user_" + userID[:8]
	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, screen_name, password_hash, role, accept_terms_version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		userID, email, screenName, "", "user", "1.0",
	)
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	return userID, email, screenName
}

func insertRefreshToken(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID, tokenID, tokenHash string, expiresAt time.Time, revokedAt *time.Time) {
	t.Helper()
	_, err := db.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_id, token_hash, expires_at, last_used_at, revoked_at, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, NOW(), $6, '127.0.0.1', 'test-agent')`,
		uuid.New().String(), userID, tokenID, tokenHash, expiresAt, revokedAt,
	)
	if err != nil {
		t.Fatalf("insertRefreshToken: %v", err)
	}
}

func countActiveTokens(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID string) int {
	t.Helper()
	var n int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		userID,
	).Scan(&n)
	if err != nil {
		t.Fatalf("countActiveTokens: %v", err)
	}
	return n
}

func isRevoked(ctx context.Context, t *testing.T, db *pgxpool.Pool, tokenID string) bool {
	t.Helper()
	var revokedAt *time.Time
	err := db.QueryRow(ctx,
		`SELECT revoked_at FROM refresh_tokens WHERE token_id = $1`,
		tokenID,
	).Scan(&revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		t.Fatalf("isRevoked: %v", err)
	}
	return revokedAt != nil
}

func TestRefresh_RotatesAndInvalidatesOldToken(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	userID, _, _ := createTestUser(ctx, t, db)

	tokenID := uuid.New().String()
	secret := "old-secret-32bytes-long-enough"
	insertRefreshToken(ctx, t, db, userID, tokenID, mustHash(secret), time.Now().UTC().Add(24*time.Hour), nil)

	svc := &Services{Dependencies: Dependencies{DB: db, JWTSecret: []byte("test-jwt-secret-key-min-32-characters")}}

	// First refresh — should succeed and rotate
	resp, err := svc.Refresh(ctx, RefreshBody{RefreshToken: tokenID + "." + secret}, &SessionMeta{IP: "127.0.0.1", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}
	if resp == nil || resp.RefreshToken == "" {
		t.Fatal("expected new refresh token")
	}

	// Old token must now be revoked
	if !isRevoked(ctx, t, db, tokenID) {
		t.Error("old token should be revoked after successful refresh")
	}

	// Reusing old token must fail (already revoked)
	_, err = svc.Refresh(ctx, RefreshBody{RefreshToken: tokenID + "." + secret}, &SessionMeta{})
	if err == nil {
		t.Fatal("reusing old refresh token should fail")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok || appErr.HTTPStatus != 401 {
		t.Fatalf("expected 401 on reuse, got: %v", err)
	}
}

func TestRefresh_ReuseDetection_RevokesFamily(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	userID, _, _ := createTestUser(ctx, t, db)

	// Create two active sessions
	tokenID1 := uuid.New().String()
	secret1 := "secret-one-32bytes-long-enough"
	insertRefreshToken(ctx, t, db, userID, tokenID1, mustHash(secret1), time.Now().UTC().Add(24*time.Hour), nil)

	tokenID2 := uuid.New().String()
	secret2 := "secret-two-32bytes-long-enough"
	insertRefreshToken(ctx, t, db, userID, tokenID2, mustHash(secret2), time.Now().UTC().Add(24*time.Hour), nil)

	// Revoke token1 manually (simulate prior rotation or logout)
	_, err := db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_id = $1`, tokenID1)
	if err != nil {
		t.Fatalf("manual revoke: %v", err)
	}

	svc := &Services{Dependencies: Dependencies{DB: db, JWTSecret: []byte("test-jwt-secret-key-min-32-characters")}}

	// Present revoked token1 — reuse detection must revoke entire family
	_, err = svc.Refresh(ctx, RefreshBody{RefreshToken: tokenID1 + "." + secret1}, &SessionMeta{})
	if err == nil {
		t.Fatal("expected error for revoked token reuse")
	}

	// Both tokens must now be revoked (family revocation)
	if !isRevoked(ctx, t, db, tokenID1) {
		t.Error("token1 should be revoked")
	}
	if !isRevoked(ctx, t, db, tokenID2) {
		t.Error("token2 should also be revoked (family revocation)")
	}
}

func TestLogout_RevokesToken(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	userID, _, _ := createTestUser(ctx, t, db)

	tokenID := uuid.New().String()
	secret := "logout-secret-32bytes-long-enough"
	insertRefreshToken(ctx, t, db, userID, tokenID, mustHash(secret), time.Now().UTC().Add(24*time.Hour), nil)

	svc := &Services{Dependencies: Dependencies{DB: db, Logger: nil}}
	claims := &middleware.UserClaims{UserID: userID}

	err := svc.Logout(ctx, claims, LogoutBody{RefreshToken: tokenID + "." + secret})
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if !isRevoked(ctx, t, db, tokenID) {
		t.Error("token should be revoked after logout")
	}
}

func TestLogout_DifferentUser_NoLeak(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	ownerID, _, _ := createTestUser(ctx, t, db)
	otherID, _, _ := createTestUser(ctx, t, db)

	tokenID := uuid.New().String()
	secret := "nosecret-32bytes-long-enough"
	insertRefreshToken(ctx, t, db, ownerID, tokenID, mustHash(secret), time.Now().UTC().Add(24*time.Hour), nil)

	svc := &Services{Dependencies: Dependencies{DB: db, Logger: nil}}
	claims := &middleware.UserClaims{UserID: otherID}

	// Logout from other user must not revoke the token (no leak)
	err := svc.Logout(ctx, claims, LogoutBody{RefreshToken: tokenID + "." + secret})
	if err != nil {
		t.Fatalf("logout should be idempotent no-op for cross-user: %v", err)
	}

	if isRevoked(ctx, t, db, tokenID) {
		t.Error("token should NOT be revoked when logged out by different user")
	}
}

func TestLogoutAll_RevokesAllActiveSessions(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	userID, _, _ := createTestUser(ctx, t, db)

	// Create multiple active sessions
	for i := 0; i < 3; i++ {
		tokenID := uuid.New().String()
		secret := "secret-" + string(rune('a'+i)) + "-32bytes-long-enough"
		insertRefreshToken(ctx, t, db, userID, tokenID, mustHash(secret), time.Now().UTC().Add(24*time.Hour), nil)
	}

	if countActiveTokens(ctx, t, db, userID) != 3 {
		t.Fatal("expected 3 active tokens before logout-all")
	}

	svc := &Services{Dependencies: Dependencies{DB: db, Logger: nil}}
	claims := &middleware.UserClaims{UserID: userID}

	err := svc.LogoutAll(ctx, claims)
	if err != nil {
		t.Fatalf("logout-all failed: %v", err)
	}

	if countActiveTokens(ctx, t, db, userID) != 0 {
		t.Error("expected 0 active tokens after logout-all")
	}
}

func TestLogoutAll_Idempotent_NoActiveSessions(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	userID, _, _ := createTestUser(ctx, t, db)

	// All tokens already revoked
	tokenID := uuid.New().String()
	secret := "revoked-secret-32bytes-long-enough"
	now := time.Now().UTC()
	insertRefreshToken(ctx, t, db, userID, tokenID, mustHash(secret), time.Now().UTC().Add(24*time.Hour), &now)

	svc := &Services{Dependencies: Dependencies{DB: db, Logger: nil}}
	claims := &middleware.UserClaims{UserID: userID}

	// Should succeed without error even with no active sessions
	err := svc.LogoutAll(ctx, claims)
	if err != nil {
		t.Fatalf("logout-all should be idempotent: %v", err)
	}
}
