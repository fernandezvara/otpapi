package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter returns a middleware that limits requests per IP and per API key (if available in context).
// Limits are per-minute windows.
func RateLimiter(perIP int, perAPIKey int) gin.HandlerFunc {
	var mu sync.Mutex
	type counter struct {
		window int64
		count  int
	}
	ipCounts := map[string]*counter{}
	keyCounts := map[string]*counter{}

	getBucket := func(m map[string]*counter, k string, limit int) (int, bool) {
		now := time.Now().Unix() / 60 // minute window
		c, ok := m[k]
		if !ok || c.window != now {
			c = &counter{window: now, count: 0}
			m[k] = c
		}
		c.count++
		return c.count, c.count <= limit
	}

	return func(c *gin.Context) {
		// Always enforce per-IP if > 0
		if perIP > 0 {
			ip := c.ClientIP()
			mu.Lock()
			cnt, ok := getBucket(ipCounts, ip, perIP)
			mu.Unlock()
			if !ok {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded (ip)", "limit": perIP, "count": cnt})
				return
			}
		}

		// Enforce per-API key if the request is authenticated and limit > 0
		if perAPIKey > 0 {
			apiKeyID := c.GetString("api_key_id")
			if apiKeyID != "" {
				mu.Lock()
				cnt, ok := getBucket(keyCounts, apiKeyID, perAPIKey)
				mu.Unlock()
				if !ok {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded (api key)", "limit": perAPIKey, "count": cnt})
					return
				}
			}
		}

		c.Next()
	}
}
