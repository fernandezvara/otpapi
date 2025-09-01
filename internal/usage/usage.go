package usage

import (
	"github.com/gin-gonic/gin"
	"otp/internal/db"
)

// Record inserts a usage event if api_key_id and customer_id are present in context.
// endpoint examples: "mfa.validate", "mfa.backup_codes.consume", "mfa.register"
func Record(c *gin.Context, endpoint string, success bool) {
	apiKeyID := c.GetString("api_key_id")
	customerID := c.GetString("customer_id")
	if apiKeyID == "" || customerID == "" {
		return
	}
	_, _ = db.DB.Exec(`INSERT INTO usage_events (customer_id, api_key_id, endpoint, success) VALUES ($1, $2, $3, $4)`, customerID, apiKeyID, endpoint, success)
}
