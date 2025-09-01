package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"otp/internal/config"
	"otp/internal/db"
	"otp/internal/keys"
)

type bootstrapRequest struct {
	CompanyName string `json:"company_name"`
	Email       string `json:"email"`
	KeyName     string `json:"key_name"`
	Environment string `json:"environment"` // "test" or "live"
}

func BootstrapSeed(c *gin.Context) {
	// Protect with a simple token
	token := c.GetHeader("X-Bootstrap-Token")
	if token == "" || token != config.Get().BootstrapToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req bootstrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CompanyName == "" || req.Email == "" || req.KeyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_name, email and key_name are required"})
		return
	}
	env := req.Environment
	if env == "" {
		env = "test"
	}
	prefix := "sk_" + env + "_"

	// Create customer
	var customerID string
	err := db.DB.QueryRow(
		"INSERT INTO customers (company_name, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		req.CompanyName, req.Email, "-",
	).Scan(&customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}

	// Create API key
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

	var apiKeyID string
	err = db.DB.QueryRow(
		`INSERT INTO api_keys (customer_id, key_name, key_prefix, key_hash, key_last_four, environment) 
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, req.KeyName, prefix, keyHash, last4, env,
	).Scan(&apiKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"customer_id": customerID,
		"api_key_id":  apiKeyID,
		"api_key":     plainKey, // show only once
	})
}
