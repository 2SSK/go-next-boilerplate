// Package config loads runtime settings from the environment.
package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/2SSK/go-next-boilerplate/internal/logger"
)

// Config holds every runtime setting for the API.
type Config struct {
	// Env is "development" or "production". Production switches logs to JSON.
	Env string
	// LogLevel is LOG_LEVEL; empty means info in production, debug otherwise.
	LogLevel slog.Level
	Port     int
	// DatabaseURL is DATABASE_URL when set, otherwise built from the libpq
	// variables (PGHOST, PGUSER, PGPASSWORD, ...). See databaseURL.
	DatabaseURL string
	// DocsEnabled serves /docs and /openapi.yaml (DOCS_ENABLED, default true).
	// Turn it off where the API port is public and the docs shouldn't be.
	DocsEnabled bool
	// OIDC says which access tokens the API accepts (see internal/auth).
	OIDC OIDC
	// Temporal is where workflows run (see internal/temporal).
	Temporal Temporal
	// ShutdownTimeout bounds how long in-flight requests may drain on SIGTERM.
	ShutdownTimeout time.Duration
}

// Temporal configures the workflow engine connection.
type Temporal struct {
	Address   string // TEMPORAL_ADDRESS, host:port of the frontend
	Namespace string // TEMPORAL_NAMESPACE
	TaskQueue string // TEMPORAL_TASK_QUEUE, shared by the API and the worker
}

// OIDC configures access-token verification.
type OIDC struct {
	// Issuer is the iss claim tokens carry (OIDC_ISSUER).
	Issuer string
	// DiscoveryURL is where the API reaches the provider, if not at Issuer
	// (OIDC_DISCOVERY_URL; in Docker, Keycloak's internal address).
	DiscoveryURL string
	// Audience must be in each token's aud claim (OIDC_AUDIENCE).
	Audience string
}

// Load reads configuration from the process environment. Unset variables fall
// back to defaults that match docker-compose.yml; invalid values are errors.
func Load() (Config, error) {
	cfg := Config{
		Env:         getEnv("APP_ENV", "development"),
		DatabaseURL: databaseURL(),
	}

	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development or production, got %q", cfg.Env)
	}

	level, err := logger.ParseLevel(os.Getenv("LOG_LEVEL"), cfg.Env)
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	port, err := getEnvInt("PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	cfg.Port = port

	docs, err := getEnvBool("DOCS_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	cfg.DocsEnabled = docs

	cfg.Temporal = Temporal{
		Address:   getEnv("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		TaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "app"),
	}

	oidc, err := loadOIDC()
	if err != nil {
		return Config{}, err
	}
	cfg.OIDC = oidc

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

// loadOIDC reads the OIDC_* variables; defaults match docker-compose.yml.
func loadOIDC() (OIDC, error) {
	o := OIDC{
		Issuer:       getEnv("OIDC_ISSUER", "http://localhost:8180/realms/app"),
		DiscoveryURL: os.Getenv("OIDC_DISCOVERY_URL"),
		Audience:     getEnv("OIDC_AUDIENCE", "api"),
	}
	for name, v := range map[string]string{"OIDC_ISSUER": o.Issuer, "OIDC_DISCOVERY_URL": o.DiscoveryURL} {
		if v == "" {
			continue
		}
		if u, err := url.Parse(v); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return OIDC{}, fmt.Errorf("%s must be an http(s) URL, got %q", name, v)
		}
	}
	return o, nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid %s %q: must be true or false", key, v)
	}
	return b, nil
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

// databaseURL returns DATABASE_URL when set (hosted Postgres usually hands out
// one URL). Otherwise it assembles one from the standard libpq variables,
// which .env and docker-compose.yml share, so the password is written once
// and any character in it is escaped correctly. Defaults match
// docker-compose.yml.
func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(getEnv("PGUSER", "app"), getEnv("PGPASSWORD", "app")),
		Host:     net.JoinHostPort(getEnv("PGHOST", "localhost"), getEnv("PGPORT", "5432")),
		Path:     "/" + getEnv("PGDATABASE", "app"),
		RawQuery: url.Values{"sslmode": {getEnv("PGSSLMODE", "disable")}}.Encode(),
	}
	return u.String()
}
