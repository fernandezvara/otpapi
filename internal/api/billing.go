package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"otp/internal/audit"
	"otp/internal/config"
	"otp/internal/db"
)

// BillingWebhook ingests provider events (Stripe stub) and stores raw payload.
func BillingWebhook(c *gin.Context) {
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read error"})
		return
	}
	sig := c.GetHeader("Stripe-Signature")
	secret := config.Get().StripeWebhookSecret
	eventType := "unknown"
	var evt stripe.Event
	if secret != "" && sig != "" {
		evt, err = webhook.ConstructEvent(b, sig, secret)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
			return
		}
		eventType = string(evt.Type)
	}
	// Attempt to link event to an internal customer via stripe_customer_id
	var custID *string
	// Parse raw JSON to extract customer reference
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err == nil {
		if d, ok := raw["data"].(map[string]any); ok {
			if obj, ok := d["object"].(map[string]any); ok {
				// Try common field: object.customer (string)
				if v, ok := obj["customer"].(string); ok && v != "" {
					var id string
					if err := db.DB.QueryRow(`SELECT id FROM customers WHERE stripe_customer_id = $1`, v).Scan(&id); err == nil {
						custID = &id
					}
				} else if objType, _ := obj["object"].(string); objType == "customer" {
					// The object itself is a customer; use its id
					if v, ok := obj["id"].(string); ok && v != "" {
						var id string
						if err := db.DB.QueryRow(`SELECT id FROM customers WHERE stripe_customer_id = $1`, v).Scan(&id); err == nil {
							custID = &id
						}
					}
				}
			}
		}
	}
	if custID != nil {
		_, err = db.DB.Exec(`INSERT INTO billing_events (customer_id, provider, event_type, payload) VALUES ($1, $2, $3, $4)`, *custID, "stripe", eventType, string(b))
	} else {
		_, err = db.DB.Exec(`INSERT INTO billing_events (provider, event_type, payload) VALUES ($1, $2, $3)`, "stripe", eventType, string(b))
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "persist failed"})
		return
	}
	meta := map[string]any{"provider": "stripe", "event_type": eventType}
	if custID != nil {
		meta["linked_customer_id"] = *custID
	}
	audit.Log(c, "billing.webhook.received", meta)
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}
