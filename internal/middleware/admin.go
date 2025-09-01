package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"otp/internal/config"
)

// AdminTokenAuth authorizes requests using the X-Bootstrap-Token header.
func AdminTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Bootstrap-Token")
		if token == "" || token != config.Get().BootstrapToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
