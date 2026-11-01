package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("defaults when unset", func(t *testing.T) {
		t.Setenv("APP_ENV", "")
		t.Setenv("PORT", "")
		t.Setenv("SHUTDOWN_TIMEOUT", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Env != "development" || cfg.Port != 8080 || cfg.ShutdownTimeout != 10*time.Second {
			t.Errorf("unexpected defaults: %+v", cfg)
		}
	})

	tests := []struct {
		name string
		key  string
		val  string
	}{
		{"rejects unknown env", "APP_ENV", "staging"},
		{"rejects non-numeric port", "PORT", "http"},
		{"rejects bad duration", "SHUTDOWN_TIMEOUT", "ten"},
		{"rejects unknown log level", "LOG_LEVEL", "loud"},
		{"rejects a non-boolean DOCS_ENABLED", "DOCS_ENABLED", "sometimes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.val)
			if _, err := Load(); err == nil {
				t.Errorf("expected error for %s=%q", tt.key, tt.val)
			}
		})
	}
}
