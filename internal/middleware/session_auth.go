package middleware

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"otp/internal/db"
)

// SessionAuth authenticates a console/customer request using X-Session-Token header.
func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := c.GetHeader("X-Session-Token")
		if tok == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing session token"})
			return
		}
		var customerID string
		var expiresAt time.Time
		err := db.DB.QueryRow(`SELECT customer_id, expires_at FROM sessions WHERE token = $1 AND revoked_at IS NULL`, tok).Scan(&customerID, &expiresAt)
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth db error"})
			return
		}
		if time.Now().After(expiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}
		c.Set("customer_id", customerID)
		c.Next()
	}
}

// SessionAuthQS authenticates like SessionAuth but allows token in query string for transports
// that cannot set custom headers (e.g., EventSource for SSE). Prefers header if present.
func SessionAuthQS() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := c.GetHeader("X-Session-Token")
		if tok == "" {
			tok = c.Query("token")
		}
		if tok == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing session token"})
			return
		}
		var customerID string
		var expiresAt time.Time
		err := db.DB.QueryRow(`SELECT customer_id, expires_at FROM sessions WHERE token = $1 AND revoked_at IS NULL`, tok).Scan(&customerID, &expiresAt)
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth db error"})
			return
		}
		if time.Now().After(expiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}
		c.Set("customer_id", customerID)
		c.Next()
	}
}
