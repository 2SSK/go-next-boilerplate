package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux)

	tests := []struct {
		path     string
		wantType string
		wantBody string
	}{
		{"/openapi.yaml", "application/yaml", "openapi: 3."},
		{"/docs", "text/html; charset=utf-8", `url: "/openapi.yaml"`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body does not contain %q", tt.wantBody)
			}
		})
	}
}

func TestDocsPageKeepsTelemetryOff(t *testing.T) {
	// Scalar defaults to sending telemetry and enabling its hosted AI agent.
	for _, want := range []string{"telemetry: false", "agent: { disabled: true }"} {
		if !strings.Contains(page, want) {
			t.Errorf("docs page lost %q", want)
		}
	}
}
