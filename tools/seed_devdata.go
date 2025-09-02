//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"otp/internal/config"
	"otp/internal/db"
	"otp/internal/keys"
)

func main() {
	cfg := config.Load()
	// Allow override via SEED_DATABASE_URL
	if v := os.Getenv("SEED_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if err := db.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.DB.Close()

	email := getenv("SEED_EMAIL", "dev@example.com")
	pw := getenv("SEED_PASSWORD", "devpass123")
	company := getenv("SEED_COMPANY", "Dev Co")
	stripeID := getenv("SEED_STRIPE_CUSTOMER_ID", "cus_dev_123")

	customerID, err := ensureCustomer(email, pw, company, stripeID)
	if err != nil {
		log.Fatalf("ensureCustomer: %v", err)
	}

	sessionToken, err := ensureSession(customerID)
	if err != nil {
		log.Fatalf("ensureSession: %v", err)
	}
	fmt.Printf("Session token: %s\n", sessionToken)

	if err := seedUsage(customerID); err != nil {
		log.Fatalf("seedUsage: %v", err)
	}
	if err := seedBilling(customerID, stripeID); err != nil {
		log.Fatalf("seedBilling: %v", err)
	}
	log.Println("Seed complete")
}

func getenv(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }

func ensureCustomer(email, password, company, stripeID string) (string, error) {
	var id string
	err := db.DB.QueryRow(`SELECT id FROM customers WHERE email = $1`, stringsLower(email)).Scan(&id)
	if err == sql.ErrNoRows {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err := db.DB.QueryRow(`INSERT INTO customers (company_name, email, password_hash, subscription_tier, is_active, stripe_customer_id) VALUES ($1,$2,$3,'starter',true,$4) RETURNING id`, company, stringsLower(email), string(hash), stripeID).Scan(&id); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	// ensure stripe id
	_, _ = db.DB.Exec(`UPDATE customers SET stripe_customer_id = $1, updated_at = NOW() WHERE id = $2`, stripeID, id)
	return id, nil
}

func stringsLower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func ensureSession(customerID string) (string, error) {
	// reuse existing non-revoked session if present
	var tok string
	err := db.DB.QueryRow(`SELECT token FROM sessions WHERE customer_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1`, customerID).Scan(&tok)
	if err == nil {
		return tok, nil
	}
	t, err := keys.RandomHex(32)
	if err != nil { return "", err }
	exp := time.Now().Add(30 * 24 * time.Hour)
	if _, err := db.DB.Exec(`INSERT INTO sessions (customer_id, token, expires_at) VALUES ($1,$2,$3)`, customerID, t, exp); err != nil {
		return "", err
	}
	return t, nil
}

func seedUsage(customerID string) error {
	rand.Seed(time.Now().UnixNano())
	endpoints := []string{"/mfa/register", "/mfa/validate", "/mfa/backup_codes/consume"}
	// find an api_key for this customer, or create a dummy one if none exists
	var apiKeyID string
	err := db.DB.QueryRow(`SELECT id FROM api_keys WHERE customer_id = $1 ORDER BY created_at LIMIT 1`, customerID).Scan(&apiKeyID)
	if err == sql.ErrNoRows {
		// create a minimal key row
		keyName := "dev-key"
		keyPrefix := "sk_dev"
		keyHash := keys.HashAPIKey("dummy")
		last4 := "0000"
		if err := db.DB.QueryRow(`INSERT INTO api_keys (customer_id, key_name, key_prefix, key_hash, key_last_four, environment) VALUES ($1,$2,$3,$4,$5,'test') RETURNING id`, customerID, keyName, keyPrefix, keyHash, last4).Scan(&apiKeyID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// generate ~30 days of usage
	for d := 0; d < 30; d++ {
		dayStart := time.Now().AddDate(0, 0, -d).Truncate(24 * time.Hour)
		count := 50 + rand.Intn(150)
		for i := 0; i < count; i++ {
			created := dayStart.Add(time.Duration(rand.Intn(24*60)) * time.Minute)
			endpoint := endpoints[rand.Intn(len(endpoints))]
			success := rand.Float64() < 0.92
			if _, err := db.DB.Exec(`INSERT INTO usage_events (customer_id, api_key_id, endpoint, success, created_at) VALUES ($1,$2,$3,$4,$5)`, customerID, apiKeyID, endpoint, success, created); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedBilling(customerID, stripeCustomerID string) error {
	// Insert a few invoice.* and customer.subscription.* events
	now := time.Now()
	// invoice.paid
	invPaid := stripeInvoicePayload(stripeCustomerID, "in_123", "paid", 1200, 1200, "usd")
	if _, err := insertBillingEvent(customerID, "stripe", "invoice.paid", invPaid, now.Add(-72*time.Hour)); err != nil { return err }
	// invoice.upcoming
	invUpcoming := stripeInvoicePayload(stripeCustomerID, "in_124", "draft", 1500, 0, "usd")
	if _, err := insertBillingEvent(customerID, "stripe", "invoice.upcoming", invUpcoming, now.Add(-24*time.Hour)); err != nil { return err }
	// customer.subscription.updated
	subUpdated := stripeSubscriptionPayload(stripeCustomerID, "active", now.Add(-15*24*time.Hour), now.Add(15*24*time.Hour))
	if _, err := insertBillingEvent(customerID, "stripe", "customer.subscription.updated", subUpdated, now.Add(-2*time.Hour)); err != nil { return err }
	return nil
}

func insertBillingEvent(customerID, provider, eventType string, payload map[string]any, createdAt time.Time) (string, error) {
	b, _ := json.Marshal(map[string]any{
		"type":   eventType,
		"data":   map[string]any{"object": payload},
		"created": createdAt.Unix(),
	})
	var id string
	err := db.DB.QueryRow(`INSERT INTO billing_events (customer_id, provider, event_type, payload, created_at) VALUES ($1,$2,$3,$4,$5) RETURNING id`, customerID, provider, eventType, string(b), createdAt).Scan(&id)
	return id, err
}

func stripeInvoicePayload(customerID, invoiceID, status string, amountDue, amountPaid int64, currency string) map[string]any {
	return map[string]any{
		"object": "invoice",
		"id": invoiceID,
		"customer": customerID,
		"status": status,
		"amount_due": amountDue,
		"amount_paid": amountPaid,
		"currency": currency,
	}
}

func stripeSubscriptionPayload(customerID, status string, periodStart, periodEnd time.Time) map[string]any {
	return map[string]any{
		"object": "subscription",
		"id": "sub_123",
		"customer": customerID,
		"status": status,
		"current_period_start": periodStart.Unix(),
		"current_period_end": periodEnd.Unix(),
	}
}
