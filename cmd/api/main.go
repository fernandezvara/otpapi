package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"otp/internal/api"
	"otp/internal/config"
	"otp/internal/crypto"
	"otp/internal/db"
	"otp/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Configure encryption key
	if err := crypto.SetKey(cfg.EncryptionKey); err != nil {
		log.Fatalf("invalid ENCRYPTION_KEY: %v", err)
	}

	// Initialize database
	if err := db.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer db.DB.Close()

	// Gin setup
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())

	// Trusted proxies (affects ClientIP())
	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			log.Fatalf("failed to set trusted proxies: %v", err)
		}
	}

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Bootstrap-Token", "X-Session-Token", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
	}))

	// Rate limiting: apply per-IP globally
	if cfg.RateLimitPerIP > 0 {
		r.Use(middleware.RateLimiter(cfg.RateLimitPerIP, 0))
	}

	// Health
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	// Metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	v1 := r.Group("/api/v1")
	{
		v1.POST("/bootstrap/seed", api.BootstrapSeed)

		// Auth & sessions
		auth := v1.Group("/auth")
		{
			auth.POST("/register", api.Register)
			auth.POST("/verify_email", api.VerifyEmail)
			auth.POST("/login", api.Login)
			// logout requires session
			auth.POST("/logout", middleware.SessionAuth(), api.Logout)
			// password reset flows
			auth.POST("/password/request_reset", api.RequestPasswordReset)
			auth.POST("/password/reset", api.ResetPassword)
		}

		mfa := v1.Group("/mfa")
		mfa.Use(middleware.APIKeyAuth())
		if cfg.RateLimitPerAPIKey > 0 {
			mfa.Use(middleware.RateLimiter(0, cfg.RateLimitPerAPIKey))
		}
		{
			mfa.POST("/register", api.RegisterMFA)
			mfa.GET("/:id/qr", api.GetQRCode)
			mfa.POST("/:id", api.ValidateOTP)
			mfa.POST("/:id/disable", api.DisableMFA)
			mfa.POST("/:id/reset", api.ResetMFA)
			mfa.POST("/:id/backup_codes/regenerate", api.RegenerateBackupCodes)
			mfa.POST("/:id/backup_codes/consume", api.ConsumeBackupCode)
		}

		// API key management
		keys := v1.Group("/keys")
		keys.Use(middleware.APIKeyAuth())
		if cfg.RateLimitPerAPIKey > 0 {
			keys.Use(middleware.RateLimiter(0, cfg.RateLimitPerAPIKey))
		}
		{
			keys.POST("/", api.CreateAPIKey)
			keys.GET("/", api.ListAPIKeys)
			keys.GET("/:id/usage", api.GetAPIKeyUsage)
			keys.POST("/:id/disable", api.DisableAPIKey)
			keys.POST("/:id/rotate", api.RotateAPIKey)
		}

		// Console (session) routes for API key management
		console := v1.Group("/console")
		console.Use(middleware.SessionAuth())
		{
			// Customer-level usage summary
			console.GET("/usage/summary", api.GetCustomerUsageSummary)

			// Billing
			console.GET("/billing/events", api.ListBillingEvents)
			console.GET("/billing/summary", api.GetBillingSummary)

			ck := console.Group("/keys")
			{
				ck.POST("/", api.CreateAPIKey)
				ck.GET("/", api.ListAPIKeys)
				ck.GET("/:id/usage", api.GetAPIKeyUsage)
				ck.POST("/:id/disable", api.DisableAPIKey)
				ck.POST("/:id/rotate", api.RotateAPIKey)
			}

			cm := console.Group("/mfa")
			{
				cm.GET("/", api.ListMFAUsers)
				cm.GET("/:id/qr", api.GetQRCode)
				cm.POST("/:id/disable", api.DisableMFA)
				cm.POST("/:id/reset", api.ResetMFA)
				cm.POST("/:id/backup_codes/regenerate", api.RegenerateBackupCodes)
			}
		}

		// Realtime analytics stream (SSE) - use QS auth to support EventSource
		v1.GET("/console/analytics/stream", middleware.SessionAuthQS(), api.AnalyticsStream)

		// Customer management (admin protected)
		customers := v1.Group("/customers")
		customers.Use(middleware.AdminTokenAuth())
		{
			customers.POST("/", api.CreateCustomer)
			customers.GET("/", api.ListCustomers)
			customers.POST("/:id", api.UpdateCustomer)
			customers.POST("/:id/disable", api.DisableCustomer)
		}

		// Billing webhooks
		v1.POST("/billing/webhook", api.BillingWebhook)
	}

	// Static test page
	r.Static("/static", "./static")
	// Docs (OpenAPI + Swagger UI)
	r.Static("/docs", "./docs")
	r.GET("/", func(c *gin.Context) { c.File("./index.html") })

	port := cfg.Port
	if port == "" { port = "8080" }
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
