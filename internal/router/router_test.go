package router

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/middleware"
)

// The requests below never reach the database, so a nil pool is enough.
func testRoutes() http.Handler {
	return Routes(Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   api.ErrorErrorCode // set for error responses
		wantType   string
	}{
		{"root", http.MethodGet, "/", "", http.StatusOK, "", "application/json"},
		{"liveness", http.MethodGet, "/livez", "", http.StatusOK, "", "application/json"},
		{"spec", http.MethodGet, "/openapi.yaml", "", http.StatusOK, "", "application/yaml"},
		{"docs page", http.MethodGet, "/docs", "", http.StatusOK, "", "text/html; charset=utf-8"},
		{"unknown path", http.MethodGet, "/nope", "", http.StatusNotFound, api.ErrorErrorCodeNotFound, "application/json"},
		{"malformed JSON body", http.MethodPost, "/api/v1/todos", `{"title":`, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"},                                             // feature:todo
		{"wrong JSON type", http.MethodPost, "/api/v1/todos", `{"title":42}`, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"},                                              // feature:todo
		{"non-numeric id", http.MethodPatch, "/api/v1/todos/abc", `{}`, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"},                                                    // feature:todo
		{"body over the limit", http.MethodPost, "/api/v1/todos", `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"}, // feature:todo
	}
	h := testRoutes()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
			if rec.Header().Get(middleware.RequestIDHeader) == "" {
				t.Error("middleware chain did not set a request ID")
			}
			if tt.wantCode != "" {
				var body api.Error
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != tt.wantCode || body.Error.Message == "" {
					t.Errorf("error body = %s, want code %q and a message", rec.Body, tt.wantCode)
				}
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
