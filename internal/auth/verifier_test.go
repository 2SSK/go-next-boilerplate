package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// fakeIssuer is an in-process OIDC provider: a discovery document, a JWKS, and
// a key to sign tokens with. publicIssuer is what tokens and discovery claim to
// be, which may differ from where the server really is (as in Docker).
type fakeIssuer struct {
	srv          *httptest.Server
	key          *rsa.PrivateKey
	publicIssuer string
}

func newFakeIssuer(t *testing.T, publicIssuer string) *fakeIssuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIssuer{key: key}
	mux := http.NewServeMux()
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	if publicIssuer == "" {
		publicIssuer = f.srv.URL + "/realms/app"
	}
	f.publicIssuer = publicIssuer

	mux.HandleFunc("GET /realms/app/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                f.publicIssuer,
			"jwks_uri":                              f.srv.URL + "/realms/app/certs",
			"authorization_endpoint":                f.publicIssuer + "/auth",
			"token_endpoint":                        f.srv.URL + "/realms/app/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("GET /realms/app/certs", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"},
		}})
	})
	return f
}

func (f *fakeIssuer) discoveryURL() string { return f.srv.URL + "/realms/app" }

// token signs claims with key (the issuer's own key unless overridden).
func (f *fakeIssuer) token(t *testing.T, claims map[string]any, key *rsa.PrivateKey) string {
	t.Helper()
	if key == nil {
		key = f.key
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"))
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(claims)
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := jws.CompactSerialize()
	return s
}

func (f *fakeIssuer) claims(overrides map[string]any) map[string]any {
	c := map[string]any{
		"iss":                f.publicIssuer,
		"aud":                []string{"api", "account"},
		"sub":                "user-123",
		"exp":                time.Now().Add(5 * time.Minute).Unix(),
		"iat":                time.Now().Unix(),
		"preferred_username": "alice",
		"email":              "alice@example.com",
		"realm_access":       map[string]any{"roles": []string{"user"}},
	}
	for k, v := range overrides {
		c[k] = v
	}
	return c
}

func TestVerify(t *testing.T) {
	f := newFakeIssuer(t, "")
	v := NewVerifier(Config{Issuer: f.publicIssuer, Audience: "api"})
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{"valid", f.token(t, f.claims(nil), nil), nil},
		{"issued for another audience", f.token(t, f.claims(map[string]any{"aud": []string{"account"}}), nil), ErrInvalidToken},
		{"from another issuer", f.token(t, f.claims(map[string]any{"iss": "https://evil.example/realms/app"}), nil), ErrInvalidToken},
		{"expired", f.token(t, f.claims(map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}), nil), ErrInvalidToken},
		{"signed with another key", f.token(t, f.claims(nil), otherKey), ErrInvalidToken},
		{"not a token", "not.a.jwt", ErrInvalidToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := v.Verify(context.Background(), tt.token)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			want := Principal{Subject: "user-123", Username: "alice", Email: "alice@example.com", Roles: []string{"user"}}
			if p.Subject != want.Subject || p.Username != want.Username || p.Email != want.Email || !p.HasRole("user") || p.HasRole("admin") {
				t.Errorf("principal = %+v, want %+v", p, want)
			}
		})
	}
}

func TestVerifyWithSeparateDiscoveryURL(t *testing.T) {
	// In Docker the API reaches Keycloak at http://keycloak:8080, while tokens
	// (and the browser) say http://localhost:8180. Discovery at one address must
	// still accept tokens issued under the other.
	f := newFakeIssuer(t, "http://localhost:8180/realms/app")
	v := NewVerifier(Config{Issuer: f.publicIssuer, DiscoveryURL: f.discoveryURL(), Audience: "api"})

	if _, err := v.Verify(context.Background(), f.token(t, f.claims(nil), nil)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyRecoversWhenTheProviderComesBack(t *testing.T) {
	f := newFakeIssuer(t, "")
	down := httptest.NewServer(http.NotFoundHandler())
	down.Close() // nothing listens here any more

	v := NewVerifier(Config{Issuer: f.publicIssuer, DiscoveryURL: down.URL + "/realms/app", Audience: "api"})
	if _, err := v.Verify(context.Background(), f.token(t, f.claims(nil), nil)); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("with the provider down, err = %v, want ErrUnavailable", err)
	}

	v.cfg.DiscoveryURL = f.discoveryURL() // the provider is back
	if _, err := v.Verify(context.Background(), f.token(t, f.claims(nil), nil)); err != nil {
		t.Fatalf("after the provider came back: %v", err)
	}
}

func TestVerifierSurvivesTheFirstRequestsContext(t *testing.T) {
	// Discovery happens lazily, inside the first request. Its key set must not
	// inherit that request's context, or every later key fetch would fail.
	f := newFakeIssuer(t, "")
	v := NewVerifier(Config{Issuer: f.publicIssuer, Audience: "api"})

	ctx, cancel := context.WithCancel(context.Background())
	if _, err := v.Verify(ctx, f.token(t, f.claims(nil), nil)); err != nil {
		t.Fatalf("first Verify: %v", err)
	}
	cancel()
	if _, err := v.Verify(context.Background(), f.token(t, f.claims(nil), nil)); err != nil {
		t.Fatalf("Verify after the first request's context ended: %v", err)
	}
}
