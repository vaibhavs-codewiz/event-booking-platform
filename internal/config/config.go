// Package config loads application configuration from the environment.
//
// Configuration is provided entirely through environment variables. A local
// .env file may be used for development convenience; it is loaded best-effort
// and never overrides variables already present in the real environment
// (godotenv's default behavior). In production, the real environment is the
// only source of configuration and no .env file exists.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	// HTTP
	Port                string
	HTTPReadTimeoutSec  int
	HTTPWriteTimeoutSec int

	// Datastores
	DatabaseURL string
	RedisURL    string

	// Auth
	JWTSecret string

	// Email (Resend)
	ResendAPIKey string
	EmailFrom    string
}

// Load reads configuration from the environment. It returns an error if any
// variable required for the current phase is missing.
//
// Only the variables the application actually uses in Phase 1 are required
// (DATABASE_URL, REDIS_URL). JWT_SECRET, RESEND_API_KEY and EMAIL_FROM become
// required in later phases; they are parsed here so the shape is stable.
func Load() (*Config, error) {
	// .env is optional. In production there is no .env file.
	_ = godotenv.Load()

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		HTTPReadTimeoutSec:  getEnvInt("HTTP_READ_TIMEOUT_SEC", 10),
		HTTPWriteTimeoutSec: getEnvInt("HTTP_WRITE_TIMEOUT_SEC", 15),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            os.Getenv("REDIS_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		ResendAPIKey:        os.Getenv("RESEND_API_KEY"),
		EmailFrom:           os.Getenv("EMAIL_FROM"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("config: REDIS_URL is required")
	}
	return cfg, nil
}

// getEnv returns the environment value for key, or fallback if it is unset or
// empty.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// getEnvInt returns the environment value for key as an int, or fallback if it
// is unset, empty, or not a valid integer.
func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
