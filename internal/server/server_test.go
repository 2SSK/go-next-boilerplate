package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2SSK/boilerplate/internal/middleware"
)

func TestRoutes(t *testing.T) {
	// Only /healthz touches the database, so a nil pool is enough here.
	h := Routes(Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantType   string
	}{
		{"root", http.MethodGet, "/", http.StatusOK, "application/json"},
		{"unknown path is a JSON 404", http.MethodGet, "/nope", http.StatusNotFound, "application/json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
			if rec.Header().Get(middleware.RequestIDHeader) == "" {
				t.Error("middleware chain did not set a request ID")
			}
		})
	}
}

func TestNewSetsTimeouts(t *testing.T) {
	srv := New(9999, http.NotFoundHandler())
	if srv.Addr != ":9999" {
		t.Errorf("Addr = %q", srv.Addr)
	}
	for name, d := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	} {
		if d <= 0 {
			t.Errorf("%s is unset; slow clients could hold connections forever", name)
		}
	}
}
