package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type UserClaims struct {
	UserID   string   `json:"sub"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Username string   `json:"username"`
	jwt.RegisteredClaims
}

func Auth(jwtSecret []byte, allowedRoles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respondError(w, http.StatusUnauthorized, "Invalid authorization format")
				return
			}

			tokenString := parts[1]

			token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(*UserClaims)
			if !ok {
				respondError(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}

			if !hasRole(claims.Roles, allowedRoles) {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func hasRole(userRoles, allowedRoles []string) bool {
	for _, allowed := range allowedRoles {
		for _, userRole := range userRoles {
			if userRole == allowed {
				return true
			}
		}
	}
	return false
}

func GetUser(ctx context.Context) *UserClaims {
	user, ok := ctx.Value(UserContextKey).(*UserClaims)
	if !ok {
		return nil
	}
	return user
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(`{"title":"` + http.StatusText(status) + `","status":` + strconv.Itoa(status) + `,"detail":"` + message + `"}`))
}
