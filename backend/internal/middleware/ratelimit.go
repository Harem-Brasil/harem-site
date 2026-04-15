package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

// extractIP extracts the IP address from RemoteAddr, removing the port if present
func extractIP(remoteAddr string) string {
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

// getClientIP returns the client IP, checking X-Forwarded-For if behind a proxy
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common when behind reverse proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple are present
		if idx := strings.Index(xff, ","); idx != -1 {
			xff = xff[:idx]
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-Ip header
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If splitting fails, use as-is (might already be just IP)
		return r.RemoteAddr
	}
	return ip
}

func RateLimit(redis *redis.Client, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			clientIP := getClientIP(r)
			key := fmt.Sprintf("ratelimit:%s", clientIP)

			if redis != nil {
				pipe := redis.Pipeline()
				incr := pipe.Incr(ctx, key)
				pipe.ExpireNX(ctx, key, time.Minute)
				_, err := pipe.Exec(ctx)

				if err == nil {
					count := incr.Val()
					limit := int64(100)

					w.Header().Set("RateLimit-Limit", strconv.FormatInt(limit, 10))
					w.Header().Set("RateLimit-Remaining", strconv.FormatInt(max(0, limit-count), 10))

					if count > limit {
						w.Header().Set("Retry-After", "60")
						w.WriteHeader(http.StatusTooManyRequests)
						w.Header().Set("Content-Type", "application/problem+json")
						w.Write([]byte(`{"title":"Rate Limit Exceeded","status":429,"detail":"Too many requests"}`))
						logger.Warn("rate limit exceeded", "client_ip", clientIP, "request_id", middleware.GetReqID(ctx))
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
