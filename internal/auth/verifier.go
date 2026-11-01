// Package auth checks the access tokens the web app forwards to the API. The
// tokens come from an OIDC provider (Keycloak in development); this package
// verifies them and turns them into a Principal. It never logs anyone in:
// login and sessions belong to the web app.
package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

// ErrInvalidToken means the token is malformed, expired, or not for this API.
var ErrInvalidToken = errors.New("invalid access token")

// ErrUnavailable means the provider couldn't be reached to fetch its keys.
var ErrUnavailable = errors.New("identity provider unavailable")

// Config says which tokens to accept.
type Config struct {
	// Issuer is the iss claim tokens must carry, e.g.
	// http://localhost:8180/realms/app.
	Issuer string
	// DiscoveryURL is where the API itself reaches the provider, when that
	// differs from Issuer (in Docker: http://keycloak:8080/realms/app).
	// Empty means Issuer.
	DiscoveryURL string
	// Audience must appear in the token's aud claim, so a token issued for
	// some other application in the realm is refused.
	Audience string
}

// Principal is who made the request.
type Principal struct {
	Subject  string // stable user ID (sub)
	Username string
	Email    string
	Roles    []string // realm roles
}

// HasRole reports whether p holds role.
func (p Principal) HasRole(role string) bool {
	return slices.Contains(p.Roles, role)
}

// Verifier checks access tokens. It contacts the provider lazily, on first
// use, so the API can start before the provider is up; a failed attempt is
// retried on the next request.
type Verifier struct {
	cfg Config

	mu       sync.Mutex
	verifier *oidc.IDTokenVerifier
}

// NewVerifier returns a Verifier for cfg. It does no I/O.
func NewVerifier(cfg Config) *Verifier {
	return &Verifier{cfg: cfg}
}

// claims are the parts of a Keycloak access token the API uses.
type claims struct {
	Username    string `json:"preferred_username"`
	Email       string `json:"email"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// Verify checks raw's signature, issuer, audience, and expiry, and returns its
// principal. Errors wrap ErrInvalidToken or ErrUnavailable.
func (v *Verifier) Verify(ctx context.Context, raw string) (Principal, error) {
	tv, err := v.tokenVerifier(ctx)
	if err != nil {
		return Principal{}, err
	}
	tok, err := tv.Verify(ctx, raw)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	var c claims
	if err := tok.Claims(&c); err != nil {
		return Principal{}, fmt.Errorf("%w: claims: %w", ErrInvalidToken, err)
	}
	return Principal{Subject: tok.Subject, Username: c.Username, Email: c.Email, Roles: c.RealmAccess.Roles}, nil
}

// tokenVerifier runs discovery once and caches the result.
func (v *Verifier) tokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.verifier != nil {
		return v.verifier, nil
	}

	// The provider keeps this context for every later key refresh, so it must
	// not be the (soon cancelled) context of the request that got here first.
	pctx := context.WithoutCancel(ctx)
	discovery := v.cfg.DiscoveryURL
	if discovery == "" || discovery == v.cfg.Issuer {
		discovery = v.cfg.Issuer
	} else {
		// Reach the provider at one address while expecting it to identify as
		// another (Docker's internal hostname vs the public one in tokens).
		pctx = oidc.InsecureIssuerURLContext(pctx, v.cfg.Issuer)
	}

	provider, err := oidc.NewProvider(pctx, discovery)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery at %s: %w", ErrUnavailable, discovery, err)
	}
	v.verifier = provider.Verifier(&oidc.Config{ClientID: v.cfg.Audience})
	return v.verifier, nil
}
