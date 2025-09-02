package api

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"image/png"
	"net/http"
	"net/url"
	"strings"
    "time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/pquerna/otp/totp"
	"otp/internal/audit"
	"otp/internal/config"
	"otp/internal/crypto"
	"otp/internal/db"
	"otp/internal/usage"
)

type RegisterRequest struct {
	ID          string `json:"id" binding:"required"`
	AccountName string `json:"account_name"`
	Issuer      string `json:"issuer" binding:"required"`
}

// CreateConsoleMFAUser creates an MFA user under the authenticated customer (session auth),
// generating a new secret and backup codes. Intended for console/testing use.
type createConsoleMFARequest struct {
    ID          string `json:"id" binding:"required"`
    AccountName string `json:"account_name"`
    Issuer      string `json:"issuer"`
}

func CreateConsoleMFAUser(c *gin.Context) {
    var req createConsoleMFARequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    customerID := c.GetString("customer_id")

    // Check if user already exists
    var exists bool
    if err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM mfa_users WHERE customer_id = $1 AND user_id = $2)", customerID, req.ID).Scan(&exists); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }
    if exists {
        c.JSON(http.StatusConflict, gin.H{"error": "User already registered for MFA"})
        return
    }

    issuer := strings.TrimSpace(req.Issuer)
    if issuer == "" { issuer = config.Get().Issuer }
    accountName := strings.TrimSpace(req.AccountName)
    if accountName == "" { accountName = fmt.Sprintf("User_%s", req.ID) }

    key, err := totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: accountName, SecretSize: 32})
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP secret"}); return }
    encSecret, err := crypto.Encrypt(key.Secret())
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt secret"}); return }

    // backup codes
    backupCodes, err := generateBackupCodes()
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup codes"}); return }
    encCodes := make([]string, len(backupCodes))
    for i, bc := range backupCodes { encCodes[i], err = crypto.Encrypt(bc); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt backup codes"}); return } }

    // Insert without api_key_id (console created)
    _, err = db.DB.Exec(`INSERT INTO mfa_users (customer_id, user_id, secret_key_encrypted, backup_codes_encrypted, account_name, issuer)
        VALUES ($1, $2, $3, $4, $5, $6)`, customerID, req.ID, encSecret, pq.Array(encCodes), accountName, issuer)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store MFA data"}); return }

    audit.Log(c, "mfa.register.console", map[string]any{"user_id": req.ID, "issuer": issuer, "account_name": accountName})
    usage.Record(c, "mfa.register.console", true)
    c.JSON(http.StatusCreated, gin.H{"qr_code_url": fmt.Sprintf("/api/v1/console/mfa/%s/qr", req.ID), "backup_codes": backupCodes})
}

type mfaUserItem struct {
    UserID      string    `json:"user_id"`
    AccountName string    `json:"account_name"`
    Issuer      string    `json:"issuer"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// ListMFAUsers lists MFA users for the authenticated customer with optional search and status filter.
func ListMFAUsers(c *gin.Context) {
    customerID := c.GetString("customer_id")
    q := strings.TrimSpace(c.Query("q"))
    status := strings.TrimSpace(strings.ToLower(c.Query("status"))) // active|disabled|all

    where := "WHERE customer_id = $1"
    args := []any{customerID}
    argIdx := 2
    if q != "" {
        where += fmt.Sprintf(" AND (user_id ILIKE $%d OR COALESCE(account_name,'') ILIKE $%d OR COALESCE(issuer,'') ILIKE $%d)", argIdx, argIdx, argIdx)
        args = append(args, "%"+q+"%")
        argIdx++
    }
    switch status {
    case "active":
        where += " AND is_active = true"
    case "disabled":
        where += " AND is_active = false"
    }

    query := "SELECT user_id, COALESCE(account_name,''), COALESCE(issuer,''), is_active, created_at, updated_at FROM mfa_users " + where + " ORDER BY created_at DESC LIMIT 200"
    rows, err := db.DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }
    defer rows.Close()

    items := []mfaUserItem{}
    for rows.Next() {
        var it mfaUserItem
        if err := rows.Scan(&it.UserID, &it.AccountName, &it.Issuer, &it.IsActive, &it.CreatedAt, &it.UpdatedAt); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan error"})
            return
        }
        items = append(items, it)
    }
    c.JSON(http.StatusOK, gin.H{"data": items})
}

// DisableMFA disables MFA for a given user (soft-disable).
func DisableMFA(c *gin.Context) {
	userID := c.Param("id")
	customerID := c.GetString("customer_id")
	res, err := db.DB.Exec(`UPDATE mfa_users SET is_active = false, updated_at = NOW() WHERE customer_id = $1 AND user_id = $2 AND is_active = true`, customerID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found or already disabled"})
		return
	}
	audit.Log(c, "mfa.disable", map[string]any{"user_id": userID})
	usage.Record(c, "mfa.disable", true)
	c.JSON(http.StatusOK, gin.H{"status": "disabled"})
}

type resetMFARequest struct {
	AccountName string `json:"account_name"`
	Issuer      string `json:"issuer"`
}

// ResetMFA regenerates the TOTP secret and backup codes, re-enables the user.
func ResetMFA(c *gin.Context) {
	userID := c.Param("id")
	customerID := c.GetString("customer_id")
	var req resetMFARequest
	_ = c.ShouldBindJSON(&req)

	// get current account_name/issuer to preserve if not provided
	var accountName, issuer string
	err := db.DB.QueryRow(`SELECT COALESCE(account_name, ''), COALESCE(issuer, '') FROM mfa_users WHERE customer_id = $1 AND user_id = $2`, customerID, userID).Scan(&accountName, &issuer)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if strings.TrimSpace(req.AccountName) != "" { accountName = req.AccountName }
	if strings.TrimSpace(req.Issuer) != "" { issuer = req.Issuer }
	if strings.TrimSpace(issuer) == "" { issuer = config.Get().Issuer }

	key, err := totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: accountName, SecretSize: 32})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP secret"})
		return
	}
	encSecret, err := crypto.Encrypt(key.Secret())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt secret"})
		return
	}
	// backup codes
	backupCodes, err := generateBackupCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup codes"})
		return
	}
	encCodes := make([]string, len(backupCodes))
	for i, bc := range backupCodes { encCodes[i], err = crypto.Encrypt(bc); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt backup codes"}); return } }

	_, err = db.DB.Exec(`UPDATE mfa_users SET is_active = true, secret_key_encrypted = $1, backup_codes_encrypted = $2, used_backup_codes_encrypted = '{}', account_name = $3, issuer = $4, updated_at = NOW() WHERE customer_id = $5 AND user_id = $6`, encSecret, pq.Array(encCodes), accountName, issuer, customerID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update MFA"})
		return
	}
	audit.Log(c, "mfa.reset", map[string]any{"user_id": userID, "issuer": issuer, "account_name": accountName})
	usage.Record(c, "mfa.reset", true)
	qrPath := fmt.Sprintf("/api/v1/mfa/%s/qr", userID)
	if strings.TrimSpace(c.GetString("api_key_id")) == "" {
		qrPath = fmt.Sprintf("/api/v1/console/mfa/%s/qr", userID)
	}
	c.JSON(http.StatusOK, gin.H{"qr_code_url": qrPath, "backup_codes": backupCodes})
}

// RegenerateBackupCodes replaces backup codes and clears used list.
func RegenerateBackupCodes(c *gin.Context) {
	userID := c.Param("id")
	customerID := c.GetString("customer_id")
	backupCodes, err := generateBackupCodes()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup codes"}); return }
	encCodes := make([]string, len(backupCodes))
	for i, bc := range backupCodes { encCodes[i], err = crypto.Encrypt(bc); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt backup codes"}); return } }
	res, err := db.DB.Exec(`UPDATE mfa_users SET backup_codes_encrypted = $1, used_backup_codes_encrypted = '{}', updated_at = NOW() WHERE customer_id = $2 AND user_id = $3 AND is_active = true`, pq.Array(encCodes), customerID, userID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"}); return }
	n, _ := res.RowsAffected(); if n == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "User not found or inactive"}); return }
	audit.Log(c, "mfa.backup_codes.regenerate", map[string]any{"user_id": userID})
	usage.Record(c, "mfa.backup_codes.regenerate", true)
	c.JSON(http.StatusOK, gin.H{"backup_codes": backupCodes})
}

type consumeBackupCodeRequest struct { Code string `json:"code" binding:"required"` }

// ConsumeBackupCode validates and consumes a single backup code.
func ConsumeBackupCode(c *gin.Context) {
	userID := c.Param("id")
	customerID := c.GetString("customer_id")
	var req consumeBackupCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }

	var encCodes []sql.NullString
	var encUsed []sql.NullString
	err := db.DB.QueryRow(`SELECT backup_codes_encrypted, COALESCE(used_backup_codes_encrypted, '{}') FROM mfa_users WHERE customer_id = $1 AND user_id = $2 AND is_active = true`, customerID, userID).Scan(pq.Array(&encCodes), pq.Array(&encUsed))
	if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "User not found or inactive"}); return }
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"}); return }

	// decrypt and find match
	foundIdx := -1
	codesDecrypted := make([]string, 0, len(encCodes))
	for i, ns := range encCodes {
		if !ns.Valid { continue }
		dec, derr := crypto.Decrypt(ns.String)
		if derr != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt backup codes"}); return }
		codesDecrypted = append(codesDecrypted, dec)
		if dec == req.Code { foundIdx = i }
	}
	if foundIdx == -1 { usage.Record(c, "mfa.backup_codes.consume", false); c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or already used backup code"}); return }

	// rebuild active encrypted codes without the consumed one
	newEncCodes := make([]string, 0, len(encCodes)-1)
	for i, ns := range encCodes { if i != foundIdx && ns.Valid { newEncCodes = append(newEncCodes, ns.String) } }
	usedEnc, err := crypto.Encrypt(req.Code)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record used code"}); return }
	// build used list
	newUsed := make([]string, 0, len(encUsed)+1)
	for _, ns := range encUsed { if ns.Valid { newUsed = append(newUsed, ns.String) } }
	newUsed = append(newUsed, usedEnc)

	_, err = db.DB.Exec(`UPDATE mfa_users SET backup_codes_encrypted = $1, used_backup_codes_encrypted = $2, updated_at = NOW() WHERE customer_id = $3 AND user_id = $4`, pq.Array(newEncCodes), pq.Array(newUsed), customerID, userID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"}); return }
	audit.Log(c, "mfa.backup_codes.consume", map[string]any{"user_id": userID})
	usage.Record(c, "mfa.backup_codes.consume", true)
	c.JSON(http.StatusOK, gin.H{"status": "consumed"})
}

type ValidateRequest struct {
	OTP string `json:"otp" binding:"required"`
}

type RegisterResponse struct {
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// generateBackupCodes returns 8 random numeric backup codes of length 8
func generateBackupCodes() ([]string, error) {
	codes := make([]string, 8)
	for i := range codes {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%010d", uint64(b[0])<<32|uint64(b[1])<<24|uint64(b[2])<<16|uint64(b[3])<<8|uint64(b[4]))[:8]
	}
	return codes, nil
}

func RegisterMFA(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customerID := c.GetString("customer_id")
	apiKeyID := c.GetString("api_key_id")

	// Check if user already exists
	var exists bool
	err := db.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM mfa_users WHERE customer_id = $1 AND user_id = $2)",
		customerID, req.ID,
	).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "User already registered for MFA"})
		return
	}

	issuer := strings.TrimSpace(req.Issuer)
	if issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "issuer is required"})
		return
	}
	accountName := req.AccountName
	if strings.TrimSpace(accountName) == "" {
		accountName = fmt.Sprintf("User_%s", req.ID)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  32,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP secret"})
		return
	}

	encryptedSecret, err := crypto.Encrypt(key.Secret())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt secret"})
		return
	}

	backupCodes, err := generateBackupCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup codes"})
		return
	}
	encryptedBackupCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		encryptedBackupCodes[i], err = crypto.Encrypt(code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt backup codes"})
			return
		}
	}

	_, err = db.DB.Exec(
		`INSERT INTO mfa_users (customer_id, api_key_id, user_id, secret_key_encrypted, backup_codes_encrypted, account_name, issuer) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		customerID, apiKeyID, req.ID, encryptedSecret, pq.Array(encryptedBackupCodes), accountName, issuer,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store MFA data"})
		return
	}

	resp := RegisterResponse{
		QRCodeURL:   fmt.Sprintf("/api/v1/mfa/%s/qr", req.ID),
		BackupCodes: backupCodes,
	}
	audit.Log(c, "mfa.register", map[string]any{"user_id": req.ID, "api_key_id": apiKeyID, "issuer": issuer, "account_name": accountName})
	usage.Record(c, "mfa.register", true)
	c.JSON(http.StatusCreated, resp)
}

func GetQRCode(c *gin.Context) {
	userID := c.Param("id")
	customerID := c.GetString("customer_id")

	var encryptedSecret, accountName, issuer string
	err := db.DB.QueryRow(
		"SELECT secret_key_encrypted, COALESCE(account_name, ''), COALESCE(issuer, '') FROM mfa_users WHERE customer_id = $1 AND user_id = $2 AND is_active = true",
		customerID, userID,
	).Scan(&encryptedSecret, &accountName, &issuer)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	secret, err := crypto.Decrypt(encryptedSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt secret"})
		return
	}

	if strings.TrimSpace(issuer) == "" {
		issuer = config.Get().Issuer
	}
	if strings.TrimSpace(accountName) == "" {
		accountName = fmt.Sprintf("User_%s", userID)
	}
	var label string
	if accountName == "-" {
		label = issuer
	} else {
		label = fmt.Sprintf("%s:%s", issuer, accountName)
	}
	otpURL := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s",
		url.QueryEscape(label), secret, url.QueryEscape(issuer),
	)

	code, err := qr.Encode(otpURL, qr.M, qr.Auto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}
	scaled, err := barcode.Scale(code, 256, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scale QR code"})
		return
	}
	c.Header("Content-Type", "image/png")
	c.Status(http.StatusOK)
	audit.Log(c, "mfa.qr_code.generated", map[string]any{"user_id": userID})
	_ = png.Encode(c.Writer, scaled)
}

func ValidateOTP(c *gin.Context) {
	userID := c.Param("id")
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	customerID := c.GetString("customer_id")

	var encryptedSecret string
	err := db.DB.QueryRow(
		"SELECT secret_key_encrypted FROM mfa_users WHERE customer_id = $1 AND user_id = $2 AND is_active = true",
		customerID, userID,
	).Scan(&encryptedSecret)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	secret, err := crypto.Decrypt(encryptedSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt secret"})
		return
	}

	valid := totp.Validate(req.OTP, secret)
	if valid {
		_, _ = db.DB.Exec("UPDATE mfa_users SET updated_at = NOW() WHERE customer_id = $1 AND user_id = $2", customerID, userID)
		audit.Log(c, "mfa.validate.success", map[string]any{"user_id": userID})
		usage.Record(c, "mfa.validate", true)
		c.JSON(http.StatusOK, gin.H{"valid": true, "message": "OTP is valid"})
		return
	}
	audit.Log(c, "mfa.validate.failure", map[string]any{"user_id": userID})
	usage.Record(c, "mfa.validate", false)
	c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "message": "Invalid OTP"})
}
