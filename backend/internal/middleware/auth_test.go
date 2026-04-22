package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestGinAuthMiddleware(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	validToken, _ := token.SignedString(secret)

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		authHeader string
		allowed    []string
		wantStatus int
	}{
		{name: "missing auth header", authHeader: "", allowed: []string{"user"}, wantStatus: http.StatusUnauthorized},
		{name: "invalid format", authHeader: "InvalidFormat", allowed: []string{"user"}, wantStatus: http.StatusUnauthorized},
		{name: "bearer prefix only", authHeader: "Bearer ", allowed: []string{"user"}, wantStatus: http.StatusUnauthorized},
		{name: "valid token with allowed role", authHeader: "Bearer " + validToken, allowed: []string{"user"}, wantStatus: http.StatusOK},
		{name: "role not in allowed list", authHeader: "Bearer " + validToken, allowed: []string{"admin"}, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(GinAuth(secret, tt.allowed, nil))
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHasRoleTable(t *testing.T) {
	tests := []struct {
		name        string
		userRoles   []string
		allowed     []string
		wantAllowed bool
	}{
		{name: "user has allowed role", userRoles: []string{"user"}, allowed: []string{"user", "admin"}, wantAllowed: true},
		{name: "user missing role", userRoles: []string{"user"}, allowed: []string{"admin"}, wantAllowed: false},
		{name: "multiple user roles match", userRoles: []string{"user", "creator"}, allowed: []string{"creator"}, wantAllowed: true},
		{name: "empty user roles", userRoles: []string{}, allowed: []string{"user"}, wantAllowed: false},
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(-time.Hour).Unix(),
	})
	expiredToken, _ := token.SignedString(secret)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuth(secret, []string{"user"}, nil))
	r.GET("/t", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expired token should be rejected: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWrongSecret(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")
	wrongSecret := []byte("wrong-secret-key-min-32-characters")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	invalidToken, _ := token.SignedString(wrongSecret)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuth(secret, []string{"user"}, nil))
	r.GET("/t", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong secret: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminOnlyAccess(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	userToken, _ := token.SignedString(secret)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuth(secret, []string{"admin"}, nil))
	r.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("user token should not access admin route: got %d, want %d", w.Code, http.StatusForbidden)
	}
}
