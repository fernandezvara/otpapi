package api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"otp/internal/db"
)

type createCustomerRequest struct {
	CompanyName     string `json:"company_name" binding:"required"`
	Email           string `json:"email" binding:"required"`
	SubscriptionTier string `json:"subscription_tier"`
}

type updateCustomerRequest struct {
	CompanyName      *string `json:"company_name"`
	Email            *string `json:"email"`
	SubscriptionTier *string `json:"subscription_tier"`
}

type customerItem struct {
	ID               string `json:"id"`
	CompanyName      string `json:"company_name"`
	Email            string `json:"email"`
	SubscriptionTier string `json:"subscription_tier"`
	IsActive         bool   `json:"is_active"`
}

func CreateCustomer(c *gin.Context) {
	var req createCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.SubscriptionTier) == "" { req.SubscriptionTier = "starter" }
	var id string
	err := db.DB.QueryRow(`INSERT INTO customers (company_name, email, password_hash, subscription_tier, is_active) VALUES ($1, $2, $3, $4, true) RETURNING id`,
		req.CompanyName, req.Email, "-", req.SubscriptionTier,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func ListCustomers(c *gin.Context) {
	// simple pagination
	limit := 50
	offset := 0
	rows, err := db.DB.Query(`SELECT id, company_name, email, subscription_tier, COALESCE(is_active, true) FROM customers ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	items := []customerItem{}
	for rows.Next() {
		var it customerItem
		if err := rows.Scan(&it.ID, &it.CompanyName, &it.Email, &it.SubscriptionTier, &it.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	var req updateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// fetch current
	var current createCustomerRequest
	err := db.DB.QueryRow(`SELECT company_name, email, subscription_tier FROM customers WHERE id = $1`, id).Scan(&current.CompanyName, &current.Email, &current.SubscriptionTier)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch failed"})
		return
	}
	if req.CompanyName != nil { current.CompanyName = *req.CompanyName }
	if req.Email != nil { current.Email = *req.Email }
	if req.SubscriptionTier != nil { current.SubscriptionTier = *req.SubscriptionTier }
	_, err = db.DB.Exec(`UPDATE customers SET company_name = $1, email = $2, subscription_tier = $3, updated_at = NOW() WHERE id = $4`, current.CompanyName, current.Email, current.SubscriptionTier, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func DisableCustomer(c *gin.Context) {
	id := c.Param("id")
	res, err := db.DB.Exec(`UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = $1 AND COALESCE(is_active, true) = true`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "disable failed"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "customer not found or already disabled"}); return }
	c.JSON(http.StatusOK, gin.H{"status": "disabled"})
}
