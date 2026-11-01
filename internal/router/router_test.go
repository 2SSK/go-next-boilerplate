package router

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/auth"
	"github.com/2SSK/go-next-boilerplate/internal/middleware"
)

// acceptAll signs in anyone presenting "Bearer good" as a user.
type acceptAll struct{}

func (acceptAll) Verify(_ context.Context, raw string) (auth.Principal, error) {
	if raw != "good" {
		return auth.Principal{}, auth.ErrInvalidToken
	}
	return auth.Principal{Subject: "u1", Roles: []string{"user"}}, nil
}

// The requests below never reach the database, so a nil pool is enough.
func testRoutes(docs bool) http.Handler {
	return Routes(Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DocsEnabled: docs, Auth: acceptAll{}})
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		signedIn   bool
		wantStatus int
		wantCode   api.ErrorErrorCode // set for error responses
		wantType   string
	}{
		{"root", http.MethodGet, "/", "", false, http.StatusOK, "", "application/json"},
		{"liveness", http.MethodGet, "/livez", "", false, http.StatusOK, "", "application/json"},
		{"spec", http.MethodGet, "/openapi.yaml", "", false, http.StatusOK, "", "application/yaml"},
		{"docs page", http.MethodGet, "/docs", "", false, http.StatusOK, "", "text/html; charset=utf-8"},
		{"unknown path", http.MethodGet, "/nope", "", false, http.StatusNotFound, api.ErrorErrorCodeNotFound, "application/json"},
		{"secured operation without a token", http.MethodGet, "/api/v1/todos", "", false, http.StatusUnauthorized, api.ErrorErrorCodeUnauthorized, "application/json"},                                    // feature:todo
		{"malformed JSON body", http.MethodPost, "/api/v1/todos", `{"title":`, true, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"},                                             // feature:todo
		{"wrong JSON type", http.MethodPost, "/api/v1/todos", `{"title":42}`, true, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"},                                              // feature:todo
		{"non-numeric id", http.MethodPatch, "/api/v1/todos/abc", `{}`, true, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"},                                                    // feature:todo
		{"body over the limit", http.MethodPost, "/api/v1/todos", `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, true, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, "application/json"}, // feature:todo
	}
	h := testRoutes(true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.signedIn {
				req.Header.Set("Authorization", "Bearer good")
			}
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
			checkEveryResponseHeaders(t, rec)
			if tt.wantCode != "" {
				var body api.Error
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != tt.wantCode || body.Error.Message == "" {
					t.Errorf("error body = %s, want code %q and a message", rec.Body, tt.wantCode)
				}
			}
		})
	}
}

// checkEveryResponseHeaders asserts what the middleware chain adds to every
// response, whatever the route.
func checkEveryResponseHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Header().Get(middleware.RequestIDHeader) == "" {
		t.Error("middleware chain did not set a request ID")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("security headers missing")
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

func TestDocsCanBeTurnedOff(t *testing.T) {
	h := testRoutes(false)
	for _, path := range []string{"/docs", "/openapi.yaml"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d with docs off, want 404", path, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if strings.Contains(rec.Body.String(), "docs") {
		t.Errorf("root still links the docs: %s", rec.Body)
	}
}

func TestJSONResponsesDenyEverything(t *testing.T) {
	rec := httptest.NewRecorder()
	testRoutes(true).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := rec.Header().Get("Content-Security-Policy"); got != "default-src 'none'; frame-ancestors 'none'" {
		t.Errorf("JSON CSP = %q", got)
	}
}
