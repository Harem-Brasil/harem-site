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

// trustedProxies contains IPs of known reverse proxies that can set X-Forwarded-For
var trustedProxies = []string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}

// isTrustedProxy checks if the remote IP is in the trusted proxies list
func isTrustedProxy(remoteIP string) bool {
	for _, trusted := range trustedProxies {
		if strings.Contains(trusted, "/") {
			// CIDR range check (simplified)
			if strings.HasPrefix(remoteIP, trusted[:strings.Index(trusted, ".")+1]) {
				return true
			}
		} else if remoteIP == trusted {
			return true
		}
	}
	return false
}

// getClientIP returns the client IP, validating X-Forwarded-For only from trusted proxies
func getClientIP(r *http.Request) string {
	// Get the remote IP (the immediate connection)
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// Only trust X-Forwarded-For headers from trusted proxies
	if isTrustedProxy(remoteIP) {
		// Check X-Forwarded-For header
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			// Take the first IP (closest to the client)
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
	}

	return remoteIP
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
