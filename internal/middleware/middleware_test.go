package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = GetRequestID(r.Context())
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

func TestLoggerRecordsRouteAndStatus(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := Chain(mux, RequestID, Logger(log))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/7", nil))

	line := buf.String()
	for _, want := range []string{`route="GET /items/{id}"`, "status=418", "level=WARN", "request_id="} {
		if !strings.Contains(line, want) {
			t.Errorf("log line missing %s: %s", want, line)
		}
	}
}

func TestRecover(t *testing.T) {
	log := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
