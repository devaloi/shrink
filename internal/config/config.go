// Package config provides application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration values.
type Config struct {
	Port        int
	DatabaseURL string
	BaseURL     string
	RateLimit   float64
	RateBurst   int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        8080,
		DatabaseURL: "./shrink.db",
		BaseURL:     "http://localhost:8080",
		RateLimit:   10,
		RateBurst:   20,
	}

	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		if p < 1 || p > 65535 {
			return nil, fmt.Errorf("PORT must be between 1 and 65535")
		}
		cfg.Port = p
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}

	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	if rateLimit := os.Getenv("RATE_LIMIT"); rateLimit != "" {
		r, err := strconv.ParseFloat(rateLimit, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT: %w", err)
		}
		if r <= 0 {
			return nil, fmt.Errorf("RATE_LIMIT must be positive")
		}
		cfg.RateLimit = r
	}

	if rateBurst := os.Getenv("RATE_BURST"); rateBurst != "" {
		b, err := strconv.Atoi(rateBurst)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_BURST: %w", err)
		}
		if b < 1 {
			return nil, fmt.Errorf("RATE_BURST must be at least 1")
		}
		cfg.RateBurst = b
	}

	return cfg, nil
}

// Addr returns the server address in host:port format.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
