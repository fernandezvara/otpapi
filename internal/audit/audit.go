package audit

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"otp/internal/db"
	"otp/internal/realtime"
)

// Log writes an audit event. actor_type: api_key|customer|system
func Log(c *gin.Context, event string, metadata map[string]any) {
	actorType := "system"
	actorID := ""
	if v, ok := c.Get("api_key_id"); ok {
		actorType = "api_key"
		actorID, _ = v.(string)
	} else if v, ok := c.Get("customer_id"); ok {
		actorType = "customer"
		actorID, _ = v.(string)
	}
	ip := c.ClientIP()
	var metaJSON []byte
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metaJSON = b
		}
	}
	if metaJSON == nil {
		metaJSON = []byte("{}")
	}
	// Attach customer_id if present in context
	customerID := ""
	if v, ok := c.Get("customer_id"); ok {
		customerID, _ = v.(string)
	}
	_, err := db.DB.Exec(
		`INSERT INTO audit_logs (customer_id, actor_type, actor_id, event, ip, metadata)
		 VALUES (NULLIF($1,'' )::uuid, $2, $3, $4, $5, $6::jsonb)`,
		customerID, actorType, actorID, event, ip, string(metaJSON),
	)
	if err != nil {
		log.Printf("audit log insert failed: %v", err)
	}

	// Fire-and-forget realtime notification scoped to customer if available
	if customerID != "" {
		realtime.PublishDefault(customerID, realtime.Event{
			Type:      "audit",
			Timestamp: time.Now(),
			Data: map[string]any{
				"event":      event,
				"actor_type": actorType,
				"actor_id":   actorID,
				"ip":         ip,
			},
		})
	}
}
