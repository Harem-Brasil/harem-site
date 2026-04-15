package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func RateLimit(redis *redis.Client, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			key := fmt.Sprintf("ratelimit:%s", r.RemoteAddr)

			if redis != nil {
				pipe := redis.Pipeline()
				incr := pipe.Incr(ctx, key)
				pipe.Expire(ctx, key, time.Minute)
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
						logger.Warn("rate limit exceeded", "remote_addr", r.RemoteAddr, "request_id", middleware.GetReqID(ctx))
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
