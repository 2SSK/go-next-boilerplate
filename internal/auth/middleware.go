package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/httpx"
)

// TokenVerifier turns a raw access token into a Principal. *Verifier is the
// real one; tests use a fake.
type TokenVerifier interface {
	Verify(ctx context.Context, raw string) (Principal, error)
}

type principalKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the caller set by Middleware. ok is false on routes the
// spec leaves public.
func PrincipalFrom(ctx context.Context) (p Principal, ok bool) {
	p, ok = ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// Middleware enforces api/openapi.yaml's security requirements. The generated
// code puts each secured operation's required roles on the request context
// (api.BearerAuthScopes) before this runs; operations without a requirement
// pass straight through. A secured operation needs a valid bearer token whose
// realm roles include every one listed.
func Middleware(v TokenVerifier, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, secured := r.Context().Value(api.BearerAuthScopes).([]string)
			if !secured {
				next.ServeHTTP(w, r)
				return
			}

			raw, ok := bearerToken(r)
			if !ok {
				unauthorized(w, "missing bearer token")
				return
			}
			p, err := v.Verify(r.Context(), raw)
			switch {
			case errors.Is(err, ErrUnavailable):
				log.ErrorContext(r.Context(), "cannot verify tokens", "error", err)
				httpx.Error(w, http.StatusServiceUnavailable, api.ErrorErrorCodeUnavailable, "authentication is temporarily unavailable")
				return
			case err != nil:
				// The reason is logged, never returned: it would help an attacker.
				log.InfoContext(r.Context(), "token rejected", "error", err)
				unauthorized(w, "invalid or expired token")
				return
			}

			for _, role := range roles {
				if !p.HasRole(role) {
					log.InfoContext(r.Context(), "role missing", "subject", p.Subject, "role", role)
					httpx.Error(w, http.StatusForbidden, api.ErrorErrorCodeForbidden, "requires role "+role)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	return token, ok && strings.EqualFold(scheme, "Bearer") && token != ""
}

// unauthorized answers 401 with the RFC 6750 challenge clients expect.
func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	httpx.Error(w, http.StatusUnauthorized, api.ErrorErrorCodeUnauthorized, message)
}
