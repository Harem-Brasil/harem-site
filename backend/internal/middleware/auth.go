package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/harem-brasil/backend/internal/utils"
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

// GinAuth valida JWT (HS256), roles e coloca claims no contexto Gin e request.
func GinAuth(jwtSecret []byte, allowedRoles []string, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if logger != nil {
				logger.Warn("authorization failure",
					"reason", "missing_header",
					"path", c.FullPath(),
					"request_id", GetRequestID(c),
					"remote_ip", c.ClientIP(),
				)
			}
			utils.RespondProblem(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "Missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			if logger != nil {
				logger.Warn("authorization failure",
					"reason", "invalid_format",
					"path", c.FullPath(),
					"request_id", GetRequestID(c),
				)
			}
			utils.RespondProblem(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "Invalid authorization format")
			c.Abort()
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
			if logger != nil {
				logger.Warn("authorization failure",
					"reason", "invalid_token",
					"path", c.FullPath(),
					"request_id", GetRequestID(c),
				)
			}
			utils.RespondProblem(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*UserClaims)
		if !ok {
			utils.RespondProblem(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "Invalid token claims")
			c.Abort()
			return
		}

		if !hasRole(claims.Roles, allowedRoles) {
			if logger != nil {
				logger.Warn("authorization failure",
					"reason", "insufficient_permissions",
					"path", c.FullPath(),
					"request_id", GetRequestID(c),
					"user_id", claims.UserID,
				)
			}
			utils.RespondProblem(c, http.StatusForbidden, http.StatusText(http.StatusForbidden), "Insufficient permissions")
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), UserContextKey, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Set(string(UserContextKey), claims)
		c.Next()
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

// GetUser obtém claims do contexto HTTP (preferência: Gin).
func GetUser(ctx context.Context) *UserClaims {
	user, ok := ctx.Value(UserContextKey).(*UserClaims)
	if !ok {
		return nil
	}
	return user
}

// MustUserClaims para handlers Gin após GinAuth.
func MustUserClaims(c *gin.Context) *UserClaims {
	if v, ok := c.Get(string(UserContextKey)); ok {
		if claims, ok := v.(*UserClaims); ok {
			return claims
		}
	}
	return GetUser(c.Request.Context())
}
