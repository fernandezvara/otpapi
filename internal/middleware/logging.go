package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs structured request information and sets a correlation ID.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			b := make([]byte, 8)
			if _, err := rand.Read(b); err == nil {
				reqID = hex.EncodeToString(b)
			} else {
				reqID = "unknown"
			}
		}
		c.Writer.Header().Set("X-Request-ID", reqID)
		c.Set("request_id", reqID)

		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method
		clientIP := c.ClientIP()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		apiKeyID, _ := c.Get("api_key_id")
		customerID, _ := c.Get("customer_id")

		log.Printf("{\"level\":\"info\",\"ts\":%d,\"request_id\":\"%s\",\"status\":%d,\"method\":\"%s\",\"path\":\"%s\",\"latency_ms\":%.3f,\"client_ip\":\"%s\",\"customer_id\":\"%v\",\"api_key_id\":\"%v\"}", time.Now().UnixNano(), reqID, status, method, path, float64(latency.Microseconds())/1000.0, clientIP, customerID, apiKeyID)
	}
}
