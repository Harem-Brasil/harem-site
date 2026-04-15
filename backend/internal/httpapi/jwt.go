package httpapi

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *Server) generateTokens(userID, email, username string, roles []string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
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
	accessToken, err = token.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshToken = uuid.New().String()

	return accessToken, refreshToken, expiresAt, nil
}
