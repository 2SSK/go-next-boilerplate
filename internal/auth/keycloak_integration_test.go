//go:build integration

package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/2SSK/go-next-boilerplate/internal/auth"
	"github.com/2SSK/go-next-boilerplate/internal/config"
)

// Runs against the Keycloak from docker-compose.yml (make kc-up), using the
// dev realm's password-login client and users.
func TestVerifyRealKeycloakTokens(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	token := func(user string) string {
		resp, err := http.PostForm(cfg.OIDC.Issuer+"/protocol/openid-connect/token", url.Values{
			"grant_type": {"password"}, "client_id": {"dev-cli"}, "username": {user}, "password": {user},
		})
		if err != nil {
			t.Skipf("keycloak unavailable: %v", err)
		}
		defer resp.Body.Close()
		var body struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.AccessToken == "" {
			t.Fatalf("no token for %s (status %d): %v", user, resp.StatusCode, err)
		}
		return body.AccessToken
	}

	v := auth.NewVerifier(auth.Config{Issuer: cfg.OIDC.Issuer, Audience: cfg.OIDC.Audience})
	for _, tt := range []struct {
		user      string
		wantAdmin bool
	}{{"alice", false}, {"admin", true}} {
		p, err := v.Verify(context.Background(), token(tt.user))
		if err != nil {
			t.Fatalf("%s: %v", tt.user, err)
		}
		if p.Username != tt.user || !p.HasRole("user") || p.HasRole("admin") != tt.wantAdmin {
			t.Errorf("%s: principal %+v", tt.user, p)
		}
	}

	// A verifier for a different audience must refuse the same tokens.
	other := auth.NewVerifier(auth.Config{Issuer: cfg.OIDC.Issuer, Audience: "some-other-api"})
	if _, err := other.Verify(context.Background(), token("alice")); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("token accepted for the wrong audience: %v", err)
	}
}
