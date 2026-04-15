package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")

	// Create valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	validToken, _ := token.SignedString(secret)

	tests := []struct {
		name       string
		authHeader string
		allowed    []string
		wantStatus int
	}{
		{
			name:       "missing auth header",
			authHeader: "",
			allowed:    []string{"user"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid format",
			authHeader: "InvalidFormat",
			allowed:    []string{"user"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bearer prefix only",
			authHeader: "Bearer ",
			allowed:    []string{"user"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token with allowed role",
			authHeader: "Bearer " + validToken,
			allowed:    []string{"user"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "role not in allowed list",
			authHeader: "Bearer " + validToken,
			allowed:    []string{"admin"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Auth(secret, tt.allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHasRole(t *testing.T) {
	tests := []struct {
		name        string
		userRoles   []string
		allowed     []string
		wantAllowed bool
	}{
		{
			name:        "user has allowed role",
			userRoles:   []string{"user"},
			allowed:     []string{"user", "admin"},
			wantAllowed: true,
		},
		{
			name:        "user missing role",
			userRoles:   []string{"user"},
			allowed:     []string{"admin"},
			wantAllowed: false,
		},
		{
			name:        "multiple user roles match",
			userRoles:   []string{"user", "creator"},
			allowed:     []string{"creator"},
			wantAllowed: true,
		},
		{
			name:        "empty user roles",
			userRoles:   []string{},
			allowed:     []string{"user"},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasRole(tt.userRoles, tt.allowed)
			if got != tt.wantAllowed {
				t.Errorf("hasRole: got %v, want %v", got, tt.wantAllowed)
			}
		})
	}
}

func TestExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")

	// Create expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(-time.Hour).Unix(),
	})
	expiredToken, _ := token.SignedString(secret)

	handler := Auth(secret, []string{"user"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expired token should be rejected: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWrongSecret(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")
	wrongSecret := []byte("wrong-secret-key-min-32-characters")

	// Create token with wrong secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	invalidToken, _ := token.SignedString(wrongSecret)

	handler := Auth(secret, []string{"user"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("token with wrong secret should be rejected: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminOnlyAccess(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")

	// Create user token (not admin)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	userToken, _ := token.SignedString(secret)

	handler := Auth(secret, []string{"admin"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("user token should not access admin route: got %d, want %d", rec.Code, http.StatusForbidden)
	}
}
