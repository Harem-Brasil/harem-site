package services

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *Services) generateTokens(userID, email, username string, roles []string) (accessToken, refreshToken, tokenID string, expiresAt time.Time, err error) {
	expiresAt = time.Now().UTC().Add(15 * time.Minute)

	claims := jwt.MapClaims{
		"sub":      userID,
		"email":    email,
		"username": username,
		"roles":    roles,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().UTC().Unix(),
		"iss":      "harem-api",
		"aud":      "harem-client",
		"type":     "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = token.SignedString(s.JWTSecret)
	if err != nil {
		return "", "", "", time.Time{}, err
	}

	tokenID = uuid.New().String()
	secret, err := generateSecureSecret()
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	refreshToken = tokenID + "." + secret
	return accessToken, refreshToken, tokenID, expiresAt, nil
}

func generateSecureSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func splitRefreshToken(refreshToken string) (tokenID, secret string, ok bool) {
	parts := strings.SplitN(refreshToken, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
