// Package config loads runtime settings from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every runtime setting for the API.
type Config struct {
	// Env is "development" or "production". Production switches logs to JSON.
	Env         string
	Port        int
	DatabaseURL string
	// ShutdownTimeout bounds how long in-flight requests may drain on SIGTERM.
	ShutdownTimeout time.Duration
}

// Load reads configuration from the process environment. Unset variables fall
// back to defaults that match docker-compose.yml; invalid values are errors.
func Load() (Config, error) {
	cfg := Config{
		Env:         getEnv("APP_ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://app:app@localhost:5432/app?sslmode=disable"),
	}

	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development or production, got %q", cfg.Env)
	}

	port, err := getEnvInt("PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	cfg.Port = port

	timeout, err := getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout = timeout

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be an integer", key, v)
	}
	return n, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be a duration like 10s", key, v)
	}
	return d, nil
}
