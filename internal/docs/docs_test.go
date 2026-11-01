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

func TestDocsPagePolicyAllowsOnlyItsOwnScript(t *testing.T) {
	rec := httptest.NewRecorder()
	servePage(rec, httptest.NewRequest(http.MethodGet, Path, nil))
	policy := rec.Header().Get("Content-Security-Policy")

	if !strings.Contains(policy, "'sha256-"+sha256Base64(initScript)+"'") {
		t.Errorf("policy does not pin the init script's hash: %s", policy)
	}
	if strings.Contains(policy, "script-src 'unsafe-inline'") || strings.Contains(policy, "script-src *") {
		t.Errorf("policy allows arbitrary scripts: %s", policy)
	}
	// The served script must be byte-identical to what was hashed.
	if !strings.Contains(rec.Body.String(), "<script>"+initScript+"</script>") {
		t.Error("the page's inline script differs from the hashed one; the browser would block it")
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
