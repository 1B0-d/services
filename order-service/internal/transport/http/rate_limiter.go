package http

import (
	"context"
	"log"
	stdhttp "net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func NewRedisRateLimiter(client *redis.Client, maxRequests int64, window time.Duration) gin.HandlerFunc {
	if client == nil || maxRequests <= 0 || window <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	windowSeconds := int64(window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		key := "rate_limit:" + c.ClientIP()
		count, err := rateLimitScript.Run(ctx, client, []string{key}, windowSeconds).Int64()
		if err != nil {
			log.Printf("rate limiter failed open for %s: %v", c.ClientIP(), err)
			c.Next()
			return
		}

		if count > maxRequests {
			c.Header("Retry-After", strconv.FormatInt(windowSeconds, 10))
			c.AbortWithStatusJSON(stdhttp.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}

		c.Next()
	}
}
