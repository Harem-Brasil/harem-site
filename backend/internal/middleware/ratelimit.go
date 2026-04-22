package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// GinRateLimit por IP cliente (Redis). Gin expõe ClientIP() com suporte a proxies configurados.
func GinRateLimit(redis *redis.Client, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		clientIP := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s", clientIP)

		if redis != nil {
			pipe := redis.Pipeline()
			incr := pipe.Incr(ctx, key)
			pipe.ExpireNX(ctx, key, time.Minute)
			_, err := pipe.Exec(ctx)

			if err == nil {
				count := incr.Val()
				limit := int64(100)

				c.Header("RateLimit-Limit", strconv.FormatInt(limit, 10))
				c.Header("RateLimit-Remaining", strconv.FormatInt(max(0, limit-count), 10))

				if count > limit {
					c.Header("Retry-After", "60")
					c.Header("Content-Type", "application/problem+json; charset=utf-8")
					if logger != nil {
						logger.Warn("rate limit exceeded",
							"client_ip", clientIP,
							"request_id", GetRequestID(c),
						)
					}
					c.JSON(http.StatusTooManyRequests, gin.H{
						"title":  "Too Many Requests",
						"status": 429,
						"detail": "Too many requests",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
