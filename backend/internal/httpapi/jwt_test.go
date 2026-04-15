package httpapi

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateTokens(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")
	s := &Server{jwtSecret: secret}

	userID := "user-123"
	email := "test@example.com"
	username := "testuser"
	roles := []string{"user"}

	accessToken, refreshToken, expiresAt, err := s.generateTokens(userID, email, username, roles)
	if err != nil {
		t.Fatalf("generateTokens failed: %v", err)
	}

	if accessToken == "" {
		t.Error("access token is empty")
	}
	if refreshToken == "" {
		t.Error("refresh token is empty")
	}
	if expiresAt.IsZero() {
		t.Error("expiresAt is zero")
	}

	// Verify access token
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		t.Errorf("failed to parse access token: %v", err)
	}

	if !token.Valid {
		t.Error("access token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims")
	}

	if claims["sub"] != userID {
		t.Errorf("user_id mismatch: got %v, want %v", claims["sub"], userID)
	}

	if claims["email"] != email {
		t.Errorf("email mismatch: got %v, want %v", claims["email"], email)
	}

	// Check token type
	if claims["type"] != "access" {
		t.Errorf("token type mismatch: got %v, want access", claims["type"])
	}
}

func TestTokenExpiry(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")
	s := &Server{jwtSecret: secret}

	accessToken, _, expiresAt, err := s.generateTokens("user-123", "test@example.com", "testuser", []string{"user"})
	if err != nil {
		t.Fatalf("generateTokens failed: %v", err)
	}

	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim not found or wrong type")
	}

	expTime := time.Unix(int64(exp), 0)
	// Allow 5 seconds tolerance for test execution time
	diff := expTime.Sub(expiresAt)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("expiry time mismatch: got %v, expected around %v", expTime, expiresAt)
	}
}

func TestInvalidSecret(t *testing.T) {
	secret := []byte("test-secret-key-min-32-characters")
	wrongSecret := []byte("wrong-secret-key-min-32-characters")
	s := &Server{jwtSecret: secret}

	accessToken, _, _, err := s.generateTokens("user-123", "test@example.com", "testuser", []string{"user"})
	if err != nil {
		t.Fatalf("generateTokens failed: %v", err)
	}

	// Try to validate with wrong secret
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return wrongSecret, nil
	})

	if err == nil && token.Valid {
		t.Error("token should be invalid with wrong secret")
	}
}
