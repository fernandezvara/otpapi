package api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"otp/internal/config"
	"otp/internal/db"
)

// GetCustomerUsageSummary returns usage summary aggregated at the customer level with optional period filter (?period=24h|7d|30d|90d|all)
func GetCustomerUsageSummary(c *gin.Context) {
	customerID := c.GetString("customer_id")
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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
		interval = "30 days"
	}

	where := "WHERE customer_id = $1"
	if interval != "" {
		where += " AND created_at >= NOW() - INTERVAL '" + interval + "'"
	}

	var total, success, failed int64
	var first, last sql.NullTime
	sumQ := "SELECT COUNT(*), COUNT(*) FILTER (WHERE success), COUNT(*) FILTER (WHERE NOT success), MIN(created_at), MAX(created_at) FROM usage_events " + where
	if err := db.DB.QueryRow(sumQ, customerID).Scan(&total, &success, &failed, &first, &last); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query error"})
		return
	}

	// by day
	byDay := []usagePoint{}
	qDay := "SELECT date_trunc('day', created_at) AS day, COUNT(*) AS total, COUNT(*) FILTER (WHERE success) AS success FROM usage_events " + where + " GROUP BY day ORDER BY day"
	rows, err := db.DB.Query(qDay, customerID)
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
	rows2, err := db.DB.Query(qEp, customerID)
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
