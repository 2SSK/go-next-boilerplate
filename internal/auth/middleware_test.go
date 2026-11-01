package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2SSK/go-next-boilerplate/api"
)

// fakeVerifier accepts "good-<role>" tokens and fails "down" with ErrUnavailable.
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, raw string) (Principal, error) {
	switch raw {
	case "good-user":
		return Principal{Subject: "u1", Roles: []string{"user"}}, nil
	case "good-admin":
		return Principal{Subject: "a1", Roles: []string{"user", "admin"}}, nil
	case "down":
		return Principal{}, ErrUnavailable
	}
	return Principal{}, ErrInvalidToken
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		roles      []string // nil: the operation is public
		header     string
		wantStatus int
		wantCode   api.ErrorErrorCode
		wantSub    string
	}{
		{"public operation needs nothing", nil, "", http.StatusOK, "", ""},
		{"secured, no header", []string{"user"}, "", http.StatusUnauthorized, api.ErrorErrorCodeUnauthorized, ""},
		{"secured, wrong scheme", []string{"user"}, "Basic dXNlcjpwYXNz", http.StatusUnauthorized, api.ErrorErrorCodeUnauthorized, ""},
		{"secured, bad token", []string{"user"}, "Bearer forged", http.StatusUnauthorized, api.ErrorErrorCodeUnauthorized, ""},
		{"secured, valid token", []string{"user"}, "Bearer good-user", http.StatusOK, "", "u1"},
		{"lower-case scheme accepted", []string{"user"}, "bearer good-user", http.StatusOK, "", "u1"},
		{"missing role", []string{"admin"}, "Bearer good-user", http.StatusForbidden, api.ErrorErrorCodeForbidden, ""},
		{"role held", []string{"admin"}, "Bearer good-admin", http.StatusOK, "", "a1"},
		{"provider down", []string{"user"}, "Bearer down", http.StatusServiceUnavailable, api.ErrorErrorCodeUnavailable, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSub string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if p, ok := PrincipalFrom(r.Context()); ok {
					gotSub = p.Subject
				}
				w.WriteHeader(http.StatusOK)
			})
			h := Middleware(fakeVerifier{}, slog.New(slog.NewTextHandler(io.Discard, nil)))(next)

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.roles != nil {
				// What the generated wrapper does for a secured operation
				req = req.WithContext(context.WithValue(req.Context(), api.BearerAuthScopes, tt.roles))
			}
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if gotSub != tt.wantSub {
				t.Errorf("principal subject = %q, want %q", gotSub, tt.wantSub)
			}
			checkAuthResponse(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}

// checkAuthResponse asserts the status, the error code, and the RFC 6750
// challenge that must accompany every 401.
func checkAuthResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode api.ErrorErrorCode) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (%s)", rec.Code, wantStatus, rec.Body)
	}
	if wantCode != "" {
		var body api.Error
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != wantCode {
			t.Errorf("body = %s, want code %q", rec.Body, wantCode)
		}
	}
	if wantStatus == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("401 without a WWW-Authenticate challenge")
	}
}

func TestPrincipalFromWithoutMiddleware(t *testing.T) {
	if _, ok := PrincipalFrom(context.Background()); ok {
		t.Error("PrincipalFrom found a principal in an empty context")
	}
}
