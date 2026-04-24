package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/harem-brasil/backend/internal/utils"
)

// rateLimitScript is a Lua script that atomically increments a key and sets
// its TTL only on the first increment, ensuring no permanent keys even on crashes.
// Compatible with Redis 6.x (no EXPIRE NX needed).
// Uses redis.NewScript so EVALSHA is used after the first call, avoiding repeated script body transfer.
var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// GinRateLimit por IP cliente (Redis). Gin expõe ClientIP() com suporte a proxies configurados.
func GinRateLimit(redis *redis.Client, logger *slog.Logger) gin.HandlerFunc {
	return ginRateLimitWithConfig(redis, logger, "ratelimit", 100, time.Minute)
}

// GinStrictRateLimit applies a stricter per-IP rate limit for sensitive auth routes (§1.3, §6.2).
func GinStrictRateLimit(redis *redis.Client, logger *slog.Logger) gin.HandlerFunc {
	return ginRateLimitWithConfig(redis, logger, "ratelimit:strict", 5, time.Minute)
}

func ginRateLimitWithConfig(rdb *redis.Client, logger *slog.Logger, keyPrefix string, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s:%s", keyPrefix, clientIP)

		if rdb != nil {
			// Lua script: atomic INCR + conditional EXPIRE (Redis 6.x compatible)
			count, err := rateLimitScript.Run(ctx, rdb, []string{key}, int(window.Seconds())).Int64()

			if err == nil {
				c.Header("RateLimit-Limit", strconv.FormatInt(limit, 10))
				c.Header("RateLimit-Remaining", strconv.FormatInt(max(0, limit-count), 10))

				if count > limit {
					c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
					if logger != nil {
						logger.Warn("rate limit exceeded",
							"client_ip", clientIP,
							"request_id", GetRequestID(c),
							"key_prefix", keyPrefix,
						)
					}
					utils.RespondProblem(c, http.StatusTooManyRequests, "Too Many Requests", "Too many requests")
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
