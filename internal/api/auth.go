package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"otp/internal/audit"
	"github.com/stripe/stripe-go/v78"
	stripeCustomer "github.com/stripe/stripe-go/v78/customer"
	"otp/internal/config"
	"otp/internal/db"
	"otp/internal/keys"
)

// Register
// POST /api/v1/auth/register
// Dev-mode: returns verification_token to facilitate testing.
type registerRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	pwHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	var customerID string
	err = db.DB.QueryRow(`INSERT INTO customers (company_name, email, password_hash, subscription_tier, is_active)
		VALUES ($1, $2, $3, 'starter', true) RETURNING id`, req.CompanyName, email, string(pwHash)).Scan(&customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not create customer (duplicate email?)"})
		return
	}
	// Optionally create Stripe customer
	cfg := config.Get()
	if cfg.StripeAPIKey != "" {
		stripe.Key = cfg.StripeAPIKey
		params := &stripe.CustomerParams{Email: stripe.String(email), Name: stripe.String(req.CompanyName)}
		if cust, cerr := stripeCustomer.New(params); cerr == nil && cust != nil {
			_, _ = db.DB.Exec(`UPDATE customers SET stripe_customer_id = $1, updated_at = NOW() WHERE id = $2`, cust.ID, customerID)
			audit.Log(c, "billing.stripe.customer_created", map[string]any{"customer_id": customerID, "stripe_customer_id": cust.ID})
		} else if cerr != nil {
			// Non-fatal: proceed without Stripe linkage
			audit.Log(c, "billing.stripe.customer_create_failed", map[string]any{"customer_id": customerID})
		}
	}
	// create email verification token (24h)
	tok, err := keys.RandomHex(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create verification token"})
		return
	}
	exp := time.Now().Add(24 * time.Hour)
	_, _ = db.DB.Exec(`INSERT INTO email_verification_tokens (customer_id, token, expires_at) VALUES ($1, $2, $3)`, customerID, tok, exp)
	audit.Log(c, "customer.register", map[string]any{"customer_id": customerID})
	c.JSON(http.StatusCreated, gin.H{"id": customerID, "verification_token": tok})
}

// VerifyEmail
// POST /api/v1/auth/verify_email
// { token }
type verifyEmailRequest struct { Token string `json:"token" binding:"required"` }

func VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	var customerID string
	var expiresAt time.Time
	var usedAt sql.NullTime
	err := db.DB.QueryRow(`SELECT customer_id, expires_at, used_at FROM email_verification_tokens WHERE token = $1`, req.Token).Scan(&customerID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"}); return }
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
	if usedAt.Valid || time.Now().After(expiresAt) { c.JSON(http.StatusBadRequest, gin.H{"error": "token expired or used"}); return }
	_, _ = db.DB.Exec(`UPDATE customers SET email_verified = true, updated_at = NOW() WHERE id = $1`, customerID)
	_, _ = db.DB.Exec(`UPDATE email_verification_tokens SET used_at = NOW() WHERE token = $1`, req.Token)
	audit.Log(c, "customer.verify_email", map[string]any{"customer_id": customerID})
	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

// Login
// POST /api/v1/auth/login
// { email, password }
type loginRequest struct { Email string `json:"email" binding:"required"`; Password string `json:"password" binding:"required"` }

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	email := strings.TrimSpace(strings.ToLower(req.Email))
	var customerID string
	var pwHash string
	var isActive sql.NullBool
	err := db.DB.QueryRow(`SELECT id, password_hash, COALESCE(is_active, true) FROM customers WHERE email = $1`, email).Scan(&customerID, &pwHash, &isActive)
	if err == sql.ErrNoRows { c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"}); return }
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
	if !isActive.Bool { c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"}); return }
	if bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(req.Password)) != nil { c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"}); return }
	// create session 30d
	tok, err := keys.RandomHex(32)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"}); return }
	exp := time.Now().Add(30 * 24 * time.Hour)
	_, err = db.DB.Exec(`INSERT INTO sessions (customer_id, token, expires_at) VALUES ($1, $2, $3)`, customerID, tok, exp)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist session"}); return }
	audit.Log(c, "customer.login", map[string]any{"customer_id": customerID})
	c.JSON(http.StatusOK, gin.H{"session_token": tok, "expires_at": exp})
}

// Logout (requires SessionAuth header)
func Logout(c *gin.Context) {
	tok := c.GetHeader("X-Session-Token")
	if tok == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"}); return }
	_, _ = db.DB.Exec(`UPDATE sessions SET revoked_at = NOW() WHERE token = $1 AND revoked_at IS NULL`, tok)
	audit.Log(c, "customer.logout", nil)
	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

// RequestPasswordReset
// POST /api/v1/auth/password/request_reset
// { email }
type requestReset struct { Email string `json:"email" binding:"required"` }

func RequestPasswordReset(c *gin.Context) {
	var req requestReset
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	email := strings.TrimSpace(strings.ToLower(req.Email))
	var customerID string
	if err := db.DB.QueryRow(`SELECT id FROM customers WHERE email = $1`, email).Scan(&customerID); err != nil {
		// do not reveal existence
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	tok, err := keys.RandomHex(24)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"}); return }
	exp := time.Now().Add(1 * time.Hour)
	_, _ = db.DB.Exec(`INSERT INTO password_reset_tokens (customer_id, token, expires_at) VALUES ($1, $2, $3)`, customerID, tok, exp)
	audit.Log(c, "customer.request_password_reset", map[string]any{"customer_id": customerID})
	// Dev-mode: return the token
	c.JSON(http.StatusOK, gin.H{"status": "ok", "reset_token": tok})
}

// ResetPassword
// POST /api/v1/auth/password/reset
// { token, new_password }
type resetPasswordReq struct { Token string `json:"token" binding:"required"`; NewPassword string `json:"new_password" binding:"required"` }

func ResetPassword(c *gin.Context) {
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	var customerID string
	var expiresAt time.Time
	var usedAt sql.NullTime
	err := db.DB.QueryRow(`SELECT customer_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`, req.Token).Scan(&customerID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"}); return }
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
	if usedAt.Valid || time.Now().After(expiresAt) { c.JSON(http.StatusBadRequest, gin.H{"error": "token expired or used"}); return }
	pwHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"}); return }
	_, err = db.DB.Exec(`UPDATE customers SET password_hash = $1, updated_at = NOW() WHERE id = $2`, string(pwHash), customerID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"}); return }
	_, _ = db.DB.Exec(`UPDATE password_reset_tokens SET used_at = NOW() WHERE token = $1`, req.Token)
	// revoke existing sessions
	_, _ = db.DB.Exec(`UPDATE sessions SET revoked_at = NOW() WHERE customer_id = $1 AND revoked_at IS NULL`, customerID)
	audit.Log(c, "customer.reset_password", map[string]any{"customer_id": customerID})
	c.JSON(http.StatusOK, gin.H{"status": "password_updated"})
}
