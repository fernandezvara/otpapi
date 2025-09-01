package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL   string
	EncryptionKey string
	Port          string
	BootstrapToken string
	Issuer        string
	// Security / networking
	CORSAllowedOrigins []string
	TrustedProxies     []string
	// Rate limiting (per minute)
	RateLimitPerIP      int
	RateLimitPerAPIKey  int
	// Billing/Stripe
	StripeAPIKey        string
	StripeWebhookSecret string
}

var cfg *Config

// Load reads environment variables and stores a global config.
func Load() *Config {
	c := &Config{
		DatabaseURL:    getenv("DATABASE_URL", "host=localhost port=5432 user=postgres password=postgres dbname=mfa_mvp sslmode=disable"),
		EncryptionKey:  getenv("ENCRYPTION_KEY", "myverysecretkey32characterslong!"),
		Port:           getenv("PORT", "8080"),
		BootstrapToken: getenv("BOOTSTRAP_TOKEN", ""),
		Issuer:         getenv("ISSUER", "SecureAuth MVP"),
		CORSAllowedOrigins: splitAndTrim(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")),
		TrustedProxies:     splitAndTrim(getenv("TRUSTED_PROXIES", "")),
		RateLimitPerIP:     getenvInt("RATE_LIMIT_PER_IP", 120),
		RateLimitPerAPIKey: getenvInt("RATE_LIMIT_PER_API_KEY", 600),
		StripeAPIKey:        getenv("STRIPE_API_KEY", ""),
		StripeWebhookSecret: getenv("STRIPE_WEBHOOK_SECRET", ""),
	}
	cfg = c
	return c
}

func Get() *Config { return cfg }

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
