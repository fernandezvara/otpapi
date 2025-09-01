package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"otp/internal/db"
	"otp/internal/keys"
)

// APIKeyAuth returns a Gin middleware that authenticates requests using a Bearer API key.
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization bearer token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		keyHash := keys.HashAPIKey(token)

		var apiKeyID, customerID string
		err := db.DB.QueryRow(
			"SELECT id, customer_id FROM api_keys WHERE key_hash = $1 AND is_active = true",
			keyHash,
		).Scan(&apiKeyID, &customerID)
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or inactive API key"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth database error"})
			return
		}

		// best-effort usage update
		_, _ = db.DB.Exec("UPDATE api_keys SET last_used_at = NOW(), usage_count = usage_count + 1 WHERE id = $1", apiKeyID)

		c.Set("api_key_id", apiKeyID)
		c.Set("customer_id", customerID)
		c.Next()
	}
}
