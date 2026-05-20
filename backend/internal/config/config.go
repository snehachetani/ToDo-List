// Package config reads application configuration from environment variables.
//
// Usage:
//
//	cfg, err := config.Load()
//	if err != nil { log.Fatal(err) }
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the application.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// Env is the runtime environment: "development" or "production".
	Env string

	// DBPath is the file path for the SQLite database.
	DBPath string

	// JWTSecret is the HMAC signing key for JWT tokens.
	// MUST be at least 32 characters in production.
	JWTSecret string

	// JWTExpiry is how long a JWT token remains valid.
	JWTExpiry time.Duration

	// AllowedOrigins is the list of origins permitted for CORS requests.
	AllowedOrigins []string

	// RateLimitRPS is the maximum requests per second per IP address.
	RateLimitRPS int
}

// Load reads environment variables and returns a validated Config.
// It returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{
		Port:      getEnv("PORT", "8080"),
		Env:       getEnv("ENV", "development"),
		DBPath:    getEnv("DB_PATH", "./todos.db"),
		JWTSecret: getEnv("JWT_SECRET", ""),
	}

	// Validate JWT secret — must be present and long enough
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(cfg.JWTSecret))
	}

	// Parse JWT expiry duration (e.g. "24h", "30m")
	expiryStr := getEnv("JWT_EXPIRY", "24h")
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY %q: %w", expiryStr, err)
	}
	cfg.JWTExpiry = expiry

	// Parse CORS origins — comma-separated list
	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:8080")
	cfg.AllowedOrigins = splitAndTrim(originsStr, ",")

	// Parse rate limit
	rpsStr := getEnv("RATE_LIMIT_RPS", "100")
	rps, err := strconv.Atoi(rpsStr)
	if err != nil || rps <= 0 {
		return nil, fmt.Errorf("invalid RATE_LIMIT_RPS %q: must be a positive integer", rpsStr)
	}
	cfg.RateLimitRPS = rps

	return cfg, nil
}

// IsDevelopment returns true when running in the development environment.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.Env, "development")
}

// Addr returns the full address string for http.ListenAndServe.
func (c *Config) Addr() string {
	return ":" + c.Port
}

// getEnv reads an environment variable, returning fallback if it is empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitAndTrim splits s by sep and trims whitespace from each element.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
