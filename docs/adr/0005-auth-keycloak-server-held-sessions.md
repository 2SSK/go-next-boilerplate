# 5. Authentication: Keycloak, server-held sessions, in-app sign-in

Status: accepted (2026-11-01), platform branch

## Context

The platform variant needs users and roles. The identity provider should be
external (Keycloak), tokens should never reach browser JavaScript, and the author
wants sign-in to happen on the app's own page rather than a redirect.

## Decision

- **Keycloak** owns users, passwords, and roles (`user`, `admin`). The realm is
  declared in `deploy/keycloak/realm-app.json` and imported at startup.
- **The API** verifies bearer tokens with go-oidc: signature, issuer, expiry, and
  **audience** (`api`), so tokens issued to other clients in the realm are refused.
  Which operations need which role is declared in `api/openapi.yaml`
  (`security: [{ bearerAuth: [user] }]`) and enforced by `internal/auth`.
- **The web server is the OAuth client.** It signs users in, keeps the tokens in an
  encrypted `httpOnly` cookie (jose, A256GCM), refreshes them in `proxy.ts`, and
  attaches them to API calls (`app/api/[...path]`, `serverApi()`). Browsers never
  hold a token. Libraries: openid-client and jose (stable); Auth.js v5 was still in
  beta.
- **Sign-in uses the app's own form** (`/login`). A Server Action sends the username
  and password to Keycloak's token endpoint (password grant, `web` client). Keycloak's
  brute-force protection locks an account after 5 failures.
- **Ownership:** resources are scoped to the token's `sub`; another user's resource
  is a 404, never a 403, so existence isn't revealed.

## Consequences

- The password grant gives up what Keycloak's own login pages provide: multi-factor
  login, password reset, social login, passkeys. To add any of them, switch the
  `web` client back to the authorization-code flow (a redirect to Keycloak with a
  themed login page); the session, proxy, and API parts stay the same.
- In Docker the web server and API reach Keycloak at an internal address while the
  issuer stays public (`AUTH_INTERNAL_ISSUER`, `OIDC_DISCOVERY_URL`). In production
  both are the same public HTTPS URL.
- `make dev` now starts Keycloak too.
