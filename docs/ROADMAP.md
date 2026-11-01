# Roadmap

Two dependable starting points, not a universal framework: a minimal core, and an extended
platform variant for distributed, authenticated applications. See
`adr/0003-core-and-platform-variants.md` for how they're managed.

## Who uses which

| Project    | Variant | Notes                                                          |
| ---------- | ------- | -------------------------------------------------------------- |
| opslens    | core    | Server monitoring (CPU, memory, disk); the first real consumer |
| shipyard   | none    | Hand-built on purpose; borrow pieces, don't retrofit           |
| tenantflow | source  | Proven Temporal, Keycloak, and Prometheus code to extract      |

## Milestone 1: core v1.0 (`main`)

The bar: `make setup → make dev → make reset → make check` is reliable, and CI proves it.

- [x] `.env` handled by the shell, not Make; one copy of the credentials (libpq `PG*`
      variables); a test proves a hostile password reaches every reader unchanged
- [x] Migrations: checksums, order check, advisory lock, `api migrate` / `make db-migrate` /
      `make db-status`; the server only verifies the schema (ADR 2)
- [x] OpenAPI contract, Go and TypeScript generated, CI drift check, `/docs` served by the API
      (ADR 1)
- [x] `/livez` and `/readyz`; one error shape with stable codes
- [x] Web app on the generated client; no hand-copied types
- [x] Playwright end-to-end tests on the production images, in CI
- [x] Compose: migrate one-shot, healthchecks, restart policies, database on localhost only
- [x] Production logging: JSON, request IDs, levels, redaction, build version
- [x] Security headers (API, docs page, web), `DOCS_ENABLED`, Dependabot
- [x] Tag `core/v1.0.0`

## Milestone 2: platform v1.0 (`platform`)

In order; each step ends with something that runs and is tested.

1. [x] Keycloak: in-app sign-in with server-held sessions, token verification (with
       audience) and role checks in Go, security schemes in the spec, per-user todos,
       e2e sign-in tests (ADR 5).
2. [x] Temporal: todo writes run as workflows on a separate `cmd/worker`, with retries,
       idempotent workflow IDs, a 503 when Temporal is down, and tests (ADR 6).
3. Observability: OpenTelemetry traces across HTTP, Postgres, and Temporal (exporter off by
   default), Prometheus metrics, Grafana behind a profile.
4. Tag `platform/v1.0.0`.

## Later, only when a project needs it

Redis (caching, rate limiting), email, seed data, a backup and restore guide.
