package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

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

type billingEventItem struct {
    ID        string          `json:"id"`
    EventType string          `json:"event_type"`
    CreatedAt time.Time       `json:"created_at"`
    Payload   json.RawMessage `json:"payload"`
}

// ListBillingEvents returns recent billing events for the authenticated customer.
// Query params: limit (default 50, max 200), type (event_type filter), since (RFC3339 timestamp)
func ListBillingEvents(c *gin.Context) {
    customerID := c.GetString("customer_id")
    if customerID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    // limit
    lim := 50
    if v := c.Query("limit"); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            if n < 1 {
                n = 1
            }
            if n > 200 {
                n = 200
            }
            lim = n
        }
    }

    // optional filters
    evType := c.Query("type")
    sinceStr := c.Query("since")
    var since *time.Time
    if sinceStr != "" {
        if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
            since = &t
        }
    }

    // build query
    where := "WHERE customer_id = $1"
    args := []any{customerID}
    argn := 2
    if evType != "" {
        where += " AND event_type = $" + strconv.Itoa(argn)
        args = append(args, evType)
        argn++
    }
    if since != nil {
        where += " AND created_at >= $" + strconv.Itoa(argn)
        args = append(args, *since)
        argn++
    }

    q := "SELECT id, event_type, created_at, payload FROM billing_events " + where + " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argn)
    args = append(args, lim)

    rows, err := db.DB.Query(q, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "query error"})
        return
    }
    defer rows.Close()

    items := []billingEventItem{}
    for rows.Next() {
        var it billingEventItem
        if err := rows.Scan(&it.ID, &it.EventType, &it.CreatedAt, &it.Payload); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
            return
        }
        items = append(items, it)
    }
    c.JSON(http.StatusOK, gin.H{"data": items})
}

type invoiceSummary struct {
    EventType string    `json:"event_type"`
    CreatedAt time.Time `json:"created_at"`
    InvoiceID string    `json:"invoice_id,omitempty"`
    AmountDue int64     `json:"amount_due,omitempty"`
    AmountPaid int64    `json:"amount_paid,omitempty"`
    Currency  string    `json:"currency,omitempty"`
    Status    string    `json:"status,omitempty"`
}

type subscriptionSummary struct {
    Status             string     `json:"status,omitempty"`
    CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
    CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
}

type billingSummary struct {
    LastInvoice  *invoiceSummary      `json:"last_invoice,omitempty"`
    Subscription *subscriptionSummary `json:"subscription,omitempty"`
}

// GetBillingSummary derives a lightweight billing summary from stored webhook events.
func GetBillingSummary(c *gin.Context) {
    customerID := c.GetString("customer_id")
    if customerID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    var sum billingSummary

    // Latest invoice.* event
    var invType string
    var invCreated time.Time
    var invPayload json.RawMessage
    err := db.DB.QueryRow(`SELECT event_type, created_at, payload FROM billing_events WHERE customer_id = $1 AND event_type LIKE 'invoice.%' ORDER BY created_at DESC LIMIT 1`, customerID).Scan(&invType, &invCreated, &invPayload)
    if err == nil {
        inv := invoiceSummary{EventType: invType, CreatedAt: invCreated}
        // Parse invoice fields if possible
        var raw map[string]any
        if json.Unmarshal(invPayload, &raw) == nil {
            if d, ok := raw["data"].(map[string]any); ok {
                if obj, ok := d["object"].(map[string]any); ok {
                    if v, _ := obj["id"].(string); v != "" { inv.InvoiceID = v }
                    if v, ok := asInt(obj["amount_due"]); ok { inv.AmountDue = v }
                    if v, ok := asInt(obj["amount_paid"]); ok { inv.AmountPaid = v }
                    if v, _ := obj["currency"].(string); v != "" { inv.Currency = v }
                    if v, _ := obj["status"].(string); v != "" { inv.Status = v }
                }
            }
        }
        sum.LastInvoice = &inv
    }

    // Latest customer.subscription.* event
    var subType string
    var subCreated time.Time
    var subPayload json.RawMessage
    err = db.DB.QueryRow(`SELECT event_type, created_at, payload FROM billing_events WHERE customer_id = $1 AND event_type LIKE 'customer.subscription.%' ORDER BY created_at DESC LIMIT 1`, customerID).Scan(&subType, &subCreated, &subPayload)
    if err == nil {
        ss := subscriptionSummary{}
        var raw map[string]any
        if json.Unmarshal(subPayload, &raw) == nil {
            if d, ok := raw["data"].(map[string]any); ok {
                if obj, ok := d["object"].(map[string]any); ok {
                    if v, _ := obj["status"].(string); v != "" { ss.Status = v }
                    if v, ok := asUnixTime(obj["current_period_start"]); ok { ss.CurrentPeriodStart = &v }
                    if v, ok := asUnixTime(obj["current_period_end"]); ok { ss.CurrentPeriodEnd = &v }
                }
            }
        }
        sum.Subscription = &ss
    }

    c.JSON(http.StatusOK, sum)
}

// Helpers
func asInt(v any) (int64, bool) {
    switch t := v.(type) {
    case float64:
        return int64(t), true
    case int64:
        return t, true
    case int:
        return int64(t), true
    default:
        return 0, false
    }
}

func asUnixTime(v any) (time.Time, bool) {
    switch t := v.(type) {
    case float64:
        return time.Unix(int64(t), 0), true
    case int64:
        return time.Unix(t, 0), true
    case int:
        return time.Unix(int64(t), 0), true
    default:
        return time.Time{}, false
    }
}
