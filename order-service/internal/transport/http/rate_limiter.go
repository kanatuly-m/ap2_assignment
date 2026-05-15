package http

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil || limit <= 0 || window <= 0 {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		key := "rate:" + c.ClientIP()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("[RateLimiter] Redis INCR failed: %v", err)
			c.Next()
			return
		}

		if count == 1 {
			if err := client.Expire(ctx, key, window).Err(); err != nil {
				log.Printf("[RateLimiter] Redis EXPIRE failed: %v", err)
			}
		}

		if count > int64(limit) {
			ttl, ttlErr := client.TTL(ctx, key).Result()
			if ttlErr == nil && ttl > 0 {
				c.Header("Retry-After", strconv.Itoa(int(ttl.Seconds())))
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":  "rate limit exceeded",
				"limit":  limit,
				"window": window.String(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
