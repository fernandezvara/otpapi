package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"otp/internal/config"
	"otp/internal/db"
	"otp/internal/middleware"
)

// setupTestRouter initializes DB and returns a Gin engine with the console routes we need.
func setupTestRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Load()
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if err := db.Init(cfg.DatabaseURL); err != nil {
		t.Skipf("skipping integration test; DB unavailable: %v (set TEST_DATABASE_URL)", err)
	}

	// Create customer, session, api key, and sample data
	custEmail := "itest@example.com"
	custID := ensureTestCustomer(t, custEmail, "itestpass", "ITest Co", "cus_itest_123")
	session := ensureTestSession(t, custID)
	apiKeyID := ensureTestAPIKey(t, custID)
	seedTestUsage(t, custID, apiKeyID)
	seedTestBilling(t, custID, "cus_itest_123")

	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	console := v1.Group("/console")
	console.Use(middleware.SessionAuth())
	{
		console.GET("/usage/summary", GetCustomerUsageSummary)
		console.GET("/billing/events", ListBillingEvents)
		console.GET("/billing/summary", GetBillingSummary)
	}
	return r, session
}

func ensureTestCustomer(t *testing.T, email, password, company, stripeID string) string {
	var id string
	err := db.DB.QueryRow(`SELECT id FROM customers WHERE email = $1`, email).Scan(&id)
	if err == sql.ErrNoRows {
		h, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err := db.DB.QueryRow(`INSERT INTO customers (company_name, email, password_hash, subscription_tier, is_active, stripe_customer_id) VALUES ($1,$2,$3,'starter',true,$4) RETURNING id`, company, email, string(h), stripeID).Scan(&id); err != nil {
			t.Fatalf("insert customer: %v", err)
		}
	} else if err != nil {
		t.Fatalf("select customer: %v", err)
	}
	_, _ = db.DB.Exec(`UPDATE customers SET stripe_customer_id = $1, updated_at = NOW() WHERE id = $2`, stripeID, id)
	return id
}

func ensureTestSession(t *testing.T, customerID string) string {
	var tok string
	err := db.DB.QueryRow(`SELECT token FROM sessions WHERE customer_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1`, customerID).Scan(&tok)
	if err == nil { return tok }

	tok = "tok_itest_" + time.Now().Format("150405")
	exp := time.Now().Add(24 * time.Hour)
	if _, err := db.DB.Exec(`INSERT INTO sessions (customer_id, token, expires_at) VALUES ($1,$2,$3)`, customerID, tok, exp); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return tok
}

func ensureTestAPIKey(t *testing.T, customerID string) string {
	var id string
	err := db.DB.QueryRow(`SELECT id FROM api_keys WHERE customer_id = $1 ORDER BY created_at LIMIT 1`, customerID).Scan(&id)
	if err == sql.ErrNoRows {
		// 64-char hex string for key_hash
		const sixtyFourHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		if err := db.DB.QueryRow(`INSERT INTO api_keys (customer_id, key_name, key_prefix, key_hash, key_last_four, environment) VALUES ($1,'itest','sk_test', $2,'beef','test') RETURNING id`, customerID, sixtyFourHex).Scan(&id); err != nil {
			t.Fatalf("insert api_key: %v", err)
		}
	} else if err != nil {
		t.Fatalf("select api_key: %v", err)
	}
	return id
}

func seedTestUsage(t *testing.T, customerID, apiKeyID string) {
	dayStart := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	for i := 0; i < 50; i++ {
		created := dayStart.Add(time.Duration(i) * time.Minute)
		if _, err := db.DB.Exec(`INSERT INTO usage_events (customer_id, api_key_id, endpoint, success, created_at) VALUES ($1,$2,$3,$4,$5)`, customerID, apiKeyID, "/mfa/validate", true, created); err != nil {
			t.Fatalf("insert usage: %v", err)
		}
	}
}

func seedTestBilling(t *testing.T, customerID, stripeCustomerID string) {
	// Minimal invoice.paid and customer.subscription.updated payloads
	inv := map[string]any{
		"type": "invoice.paid",
		"data": map[string]any{"object": map[string]any{
			"object": "invoice", "id": "in_itest_1", "customer": stripeCustomerID, "status": "paid", "amount_due": 1000.0, "amount_paid": 1000.0, "currency": "usd",
		}},
	}
	b1, _ := json.Marshal(inv)
	if _, err := db.DB.Exec(`INSERT INTO billing_events (customer_id, provider, event_type, payload, created_at) VALUES ($1,'stripe','invoice.paid',$2,NOW()-INTERVAL '1 hour')`, customerID, string(b1)); err != nil {
		t.Fatalf("insert invoice event: %v", err)
	}
	sub := map[string]any{
		"type": "customer.subscription.updated",
		"data": map[string]any{"object": map[string]any{
			"object": "subscription", "id": "sub_itest_1", "customer": stripeCustomerID, "status": "active", "current_period_start": float64(time.Now().Add(-12*time.Hour).Unix()), "current_period_end": float64(time.Now().Add(12*time.Hour).Unix()),
		}},
	}
	b2, _ := json.Marshal(sub)
	if _, err := db.DB.Exec(`INSERT INTO billing_events (customer_id, provider, event_type, payload, created_at) VALUES ($1,'stripe','customer.subscription.updated',$2,NOW()-INTERVAL '30 minutes')`, customerID, string(b2)); err != nil {
		t.Fatalf("insert sub event: %v", err)
	}
}

func doAuthedGet(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, bytes.NewReader(nil))
	req.Header.Set("X-Session-Token", token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestUsageSummaryAndBillingEndpoints(t *testing.T) {
	r, token := setupTestRouter(t)

	// Usage summary
	res := doAuthedGet(r, "/api/v1/console/usage/summary?period=7d", token)
	if res.Code != http.StatusOK {
		t.Fatalf("usage summary status=%d body=%s", res.Code, res.Body.String())
	}
	var usage struct {
		Total int64 `json:"total"`
		EstimatedCost float64 `json:"estimated_cost_usd"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &usage); err != nil {
		t.Fatalf("usage json: %v", err)
	}
	if usage.Total <= 0 {
		t.Fatalf("expected usage total > 0, got %d", usage.Total)
	}

	// Billing events
	res2 := doAuthedGet(r, "/api/v1/console/billing/events?limit=10", token)
	if res2.Code != http.StatusOK {
		t.Fatalf("billing events status=%d body=%s", res2.Code, res2.Body.String())
	}
	var events struct { Data []any `json:"data"` }
	if err := json.Unmarshal(res2.Body.Bytes(), &events); err != nil {
		t.Fatalf("events json: %v", err)
	}
	if len(events.Data) < 2 {
		t.Fatalf("expected at least 2 billing events, got %d", len(events.Data))
	}

	// Billing summary
	res3 := doAuthedGet(r, "/api/v1/console/billing/summary", token)
	if res3.Code != http.StatusOK {
		t.Fatalf("billing summary status=%d body=%s", res3.Code, res3.Body.String())
	}
	var sum struct {
		LastInvoice  map[string]any `json:"last_invoice"`
		Subscription map[string]any `json:"subscription"`
	}
	if err := json.Unmarshal(res3.Body.Bytes(), &sum); err != nil {
		t.Fatalf("summary json: %v", err)
	}
	if sum.LastInvoice == nil || sum.Subscription == nil {
		t.Fatalf("expected invoice and subscription in summary, got %+v", sum)
	}
}
