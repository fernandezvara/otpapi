package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"otp/internal/config"
	"otp/internal/db"
	"otp/internal/audit"
	"otp/internal/keys"
)

type createKeyRequest struct {
	KeyName     string `json:"key_name" binding:"required"`
	Environment string `json:"environment"` // test|live
}

type usagePoint struct {
    Day     time.Time `json:"day"`
    Total   int64     `json:"total"`
    Success int64     `json:"success"`
}

type usageByEndpoint struct {
    Endpoint string `json:"endpoint"`
    Total    int64  `json:"total"`
    Success  int64  `json:"success"`
}

type usageSummary struct {
    Total      int64              `json:"total"`
    Success    int64              `json:"success"`
    Failed     int64              `json:"failed"`
    EstimatedCostUSD float64      `json:"estimated_cost_usd"`
    FirstEvent *time.Time         `json:"first_event,omitempty"`
    LastEvent  *time.Time         `json:"last_event,omitempty"`
    ByDay      []usagePoint       `json:"by_day"`
    ByEndpoint []usageByEndpoint  `json:"by_endpoint"`
}

// GetAPIKeyUsage returns usage summary for an API key with optional period filter (?period=24h|7d|30d|90d|all)
func GetAPIKeyUsage(c *gin.Context) {
    customerID := c.GetString("customer_id")
    id := c.Param("id")

    var exists bool
    if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = $1 AND customer_id = $2)`, id, customerID).Scan(&exists); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
        return
    }

    period := strings.TrimSpace(strings.ToLower(c.Query("period")))
    interval := "30 days"
    switch period {
    case "", "30d":
        interval = "30 days"
    case "24h":
        interval = "24 hours"
    case "7d":
        interval = "7 days"
    case "90d":
        interval = "90 days"
    case "all":
        interval = ""
    default:
        // fallback to 30 days on invalid
        interval = "30 days"
    }

    where := "WHERE api_key_id = $1"
    if interval != "" {
        where += " AND created_at >= NOW() - INTERVAL '" + interval + "'"
    }

    var total, success, failed int64
    var first, last sql.NullTime
    sumQ := "SELECT COUNT(*), COUNT(*) FILTER (WHERE success), COUNT(*) FILTER (WHERE NOT success), MIN(created_at), MAX(created_at) FROM usage_events " + where
    if err := db.DB.QueryRow(sumQ, id).Scan(&total, &success, &failed, &first, &last); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "query error"})
        return
    }

    // by day
    byDay := []usagePoint{}
    qDay := "SELECT date_trunc('day', created_at) AS day, COUNT(*) AS total, COUNT(*) FILTER (WHERE success) AS success FROM usage_events " + where + " GROUP BY day ORDER BY day"
    rows, err := db.DB.Query(qDay, id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "query error"})
        return
    }
    defer rows.Close()
    for rows.Next() {
        var p usagePoint
        if err := rows.Scan(&p.Day, &p.Total, &p.Success); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
            return
        }
        byDay = append(byDay, p)
    }

    // by endpoint
    byEp := []usageByEndpoint{}
    qEp := "SELECT endpoint, COUNT(*) AS total, COUNT(*) FILTER (WHERE success) AS success FROM usage_events " + where + " GROUP BY endpoint ORDER BY total DESC LIMIT 20"
    rows2, err := db.DB.Query(qEp, id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "query error"})
        return
    }
    defer rows2.Close()
    for rows2.Next() {
        var e usageByEndpoint
        if err := rows2.Scan(&e.Endpoint, &e.Total, &e.Success); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
            return
        }
        byEp = append(byEp, e)
    }

    resp := usageSummary{Total: total, Success: success, Failed: failed, ByDay: byDay, ByEndpoint: byEp}
    if first.Valid { resp.FirstEvent = &first.Time }
    if last.Valid { resp.LastEvent = &last.Time }
    // estimated cost
    price := config.Get().PricePerRequestUSD
    resp.EstimatedCostUSD = float64(total) * price
    c.JSON(http.StatusOK, resp)
}

type apiKeyItem struct {
	ID         string    `json:"id"`
	KeyName    string    `json:"key_name"`
	KeyPrefix  string    `json:"key_prefix"`
	LastFour   string    `json:"key_last_four"`
	Environment string   `json:"environment"`
	IsActive   bool      `json:"is_active"`
	UsageCount int64     `json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateAPIKey creates a new API key for the authenticated customer and returns the plaintext key once.
func CreateAPIKey(c *gin.Context) {
	customerID := c.GetString("customer_id")
	var req createKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	env := strings.TrimSpace(req.Environment)
	if env == "" {
		env = "test"
	}
	prefix := "sk_" + env + "_"

	randHex, err := keys.RandomHex(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	plainKey := prefix + randHex
	keyHash := keys.HashAPIKey(plainKey)
	last4 := ""
	if len(plainKey) >= 4 {
		last4 = plainKey[len(plainKey)-4:]
	}

	var id string
	err = db.DB.QueryRow(
		`INSERT INTO api_keys (customer_id, key_name, key_prefix, key_hash, key_last_four, environment)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, req.KeyName, prefix, keyHash, last4, env,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api key"})
		return
	}

	audit.Log(c, "api_key.create", map[string]any{"api_key_id": id, "env": env, "key_name": req.KeyName})
	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"api_key": plainKey,
	})
}

// ListAPIKeys lists API keys for the authenticated customer.
func ListAPIKeys(c *gin.Context) {
	customerID := c.GetString("customer_id")
	rows, err := db.DB.Query(`SELECT id, key_name, key_prefix, key_last_four, environment, is_active, usage_count, created_at FROM api_keys WHERE customer_id = $1 ORDER BY created_at DESC`, customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query api keys"})
		return
	}
	defer rows.Close()
	items := []apiKeyItem{}
	for rows.Next() {
		var it apiKeyItem
		if err := rows.Scan(&it.ID, &it.KeyName, &it.KeyPrefix, &it.LastFour, &it.Environment, &it.IsActive, &it.UsageCount, &it.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// DisableAPIKey disables an API key by id for the authenticated customer.
func DisableAPIKey(c *gin.Context) {
	customerID := c.GetString("customer_id")
	id := c.Param("id")
	res, err := db.DB.Exec(`UPDATE api_keys SET is_active = false, updated_at = NOW() WHERE id = $1 AND customer_id = $2 AND is_active = true`, id, customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable api key"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found or already disabled"})
		return
	}
	audit.Log(c, "api_key.disable", map[string]any{"api_key_id": id})
	c.JSON(http.StatusOK, gin.H{"status": "disabled"})
}

// RotateAPIKey disables the specified key and creates a new one with same env and name.
func RotateAPIKey(c *gin.Context) {
	customerID := c.GetString("customer_id")
	id := c.Param("id")
	var keyName, env string
	err := db.DB.QueryRow(`SELECT key_name, environment FROM api_keys WHERE id = $1 AND customer_id = $2`, id, customerID).Scan(&keyName, &env)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch api key"})
		return
	}
	// disable old key
	_, _ = db.DB.Exec(`UPDATE api_keys SET is_active = false, updated_at = NOW() WHERE id = $1 AND customer_id = $2`, id, customerID)

	// create new key
	randHex, err := keys.RandomHex(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	prefix := "sk_" + env + "_"
	plainKey := prefix + randHex
	keyHash := keys.HashAPIKey(plainKey)
	last4 := ""
	if len(plainKey) >= 4 {
		last4 = plainKey[len(plainKey)-4:]
	}
	var newID string
	err = db.DB.QueryRow(
		`INSERT INTO api_keys (customer_id, key_name, key_prefix, key_hash, key_last_four, environment)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, keyName, prefix, keyHash, last4, env,
	).Scan(&newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rotated key"})
		return
	}
	audit.Log(c, "api_key.rotate", map[string]any{"old_api_key_id": id, "new_api_key_id": newID})
	c.JSON(http.StatusCreated, gin.H{"id": newID, "api_key": plainKey})
}
