package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2SSK/go-next-boilerplate/internal/logger"
)

func newLog(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	return logger.New(logger.Options{Env: "production", Level: level, Service: "test"}, buf)
}

func lastLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &m); err != nil {
		t.Fatalf("not JSON: %q", buf.String())
	}
	return m
}

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = logger.RequestID(r.Context())
	}))

	t.Run("generates an ID when absent", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if seen == "" || rec.Header().Get(RequestIDHeader) != seen {
			t.Errorf("context id %q, header %q", seen, rec.Header().Get(RequestIDHeader))
		}
	})

	t.Run("keeps the client's ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(RequestIDHeader, "abc-123")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if seen != "abc-123" {
			t.Errorf("id = %q, want abc-123", seen)
		}
	})
}

func TestLoggerLine(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("short and stout"))
	})
	h := Chain(mux, RequestID, Logger(newLog(&buf, slog.LevelDebug)))

	req := httptest.NewRequest(http.MethodGet, "/items/7", nil)
	req.Header.Set(RequestIDHeader, "req-1")
	req.Header.Set("User-Agent", "curl/8")
	h.ServeHTTP(httptest.NewRecorder(), req)

	line := lastLine(t, &buf)
	if ms, ok := line["duration_ms"].(float64); !ok || ms < 0 {
		t.Errorf("duration_ms = %v, want milliseconds as a number", line["duration_ms"])
	}
	for key, want := range map[string]any{
		"level":      "WARN",
		"route":      "GET /items/{id}",
		"status":     float64(418),
		"bytes":      float64(len("short and stout")),
		"request_id": "req-1",
		"user_agent": "curl/8",
	} {
		if line[key] != want {
			t.Errorf("%s = %v, want %v", key, line[key], want)
		}
	}
}

func TestLoggerQuietsHealthProbes(t *testing.T) {
	tests := []struct {
		path      string
		status    int
		wantLevel string
	}{
		{"/readyz", http.StatusOK, "DEBUG"},
		{"/livez", http.StatusOK, "DEBUG"},
		{"/readyz", http.StatusServiceUnavailable, "ERROR"}, // a failing probe is news
		{"/api/v1/things", http.StatusOK, "INFO"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var buf bytes.Buffer
			h := Logger(newLog(&buf, slog.LevelDebug))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, tt.path, nil))
			if got := lastLine(t, &buf)["level"]; got != tt.wantLevel {
				t.Errorf("level = %v, want %v", got, tt.wantLevel)
			}
		})
	}
}

func TestRecoverLogsTheStack(t *testing.T) {
	var buf bytes.Buffer
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), RequestID, Recover(newLog(&buf, slog.LevelInfo)))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}

	line := lastLine(t, &buf)
	stack, _ := line["stack"].(string)
	if line["panic"] != "boom" || !strings.Contains(stack, "middleware_test.go") || line["request_id"] == nil {
		t.Errorf("panic line = %v, want panic, a stack through this test, and a request_id", line)
	}
}

func TestBodyLimit(t *testing.T) {
	h := BodyLimit(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Body.Read(make([]byte, 16)); err == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("way too long")))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want the body read to fail past the limit", rec.Code)
	}
}
