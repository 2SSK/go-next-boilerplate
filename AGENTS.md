- Do not preserve backward compatibility. Remove obsolete paths instead of adding compatibility layers, fallbacks, or migrations.
- Choose the simplest implementation that fully meets the current requirements. Avoid speculative abstractions, configurations, and indirections.
- Grow the system in layers. Start from the smallest version that works to end, and add each new capability on top of a product that already works. Never trade a working product for unfinished complexity.
- Keep components modular and concerns clearly separated.
- Prefer established, well-maintained libraries when they reduce overall complexity or improve reliaability. Do not reimplement common functionality without a clear reason.
- Lean on the dependencies already in the project before writing your own implementation or adding packages. Do not assume a library lacks a capability without checking its documentation and types.
- Make architectural decisions for the long term. Do not acccept a stopgap that only works for now and is meant to be replaced later.

# Agent guide

A Go API (`api/`, `cmd/`, `internal/`) and a Next.js frontend (`web/`) in one repo, joined by an
OpenAPI contract (`api/openapi.yaml`). The browser only talks to Next.js, which forwards
`/api/*`, `/readyz` and `/livez` to the API. The API serves its own docs (`/docs`,
`/openapi.yaml` on :8080); they are deliberately not proxied through the web app. Web-specific rules and the
design system are in `web/AGENTS.md`; read it before touching anything under `web/`.
Its "Rendering and performance" rules are not optional: shared chrome (navbar, footer) lives
in layouts, never pages; client components stay small leaves; page data loads on the server
and streams behind `<Suspense>`.

Decisions with their reasoning are in `docs/adr/`; the plan is `docs/ROADMAP.md`. This is the
core variant (`main`); the `platform` branch extends it and nothing here may depend on it.

## Commands

Run everything through the Makefile; `make` lists the targets.

- `make dev`: Postgres, migrations, API (:8080), web (:3000). Needs `make setup` once.
- `make generate`: regenerate Go and TypeScript from `api/openapi.yaml`. Run after every spec
  edit, and commit the result.
- `make db-migrate` / `make db-status`: apply or list migrations.
- `make test`: Go unit tests (race detector) and the `.env` round-trip test.
- `make test-integration`: Go tests against Postgres (`make db-up` first).
- `make lint`: gofmt, go vet, golangci-lint, ESLint, tsc, Prettier.
- `make check`: lint + contract check + test + build. CI runs the same gates.
- `make up`, then `make e2e`: Playwright against the production images.
- `make fmt`: format Go and web code.

## Definition of done

- `make check` passes with no warnings. ESLint runs with `--max-warnings=0`.
- New behavior has tests: table-driven Go tests with subtests, one per case. A change to what a
  user can do also gets (or updates) an e2e test in `web/e2e/`.
- The spec and the generated code agree (`make generate-check`, part of `make check`).
- If you changed anything a reset touches, `make reset` still produces a tree that passes
  `make check`. CI checks this too.
- The README is updated when commands, env vars, or endpoints change. Every env var is also
  documented in `.env.example`.

## Layout

| Path                  | Holds                                                                            |
| --------------------- | -------------------------------------------------------------------------------- |
| `api/`                | `openapi.yaml` (the contract), `api.gen.go` (generated: never edit), `errors.go` |
| `cmd/api`             | Startup only: `serve` (default) and `migrate [-status]`                          |
| `internal/router`     | Mounts the generated routes, the docs, the JSON 404; builds the `http.Server`    |
| `internal/docs`       | `/openapi.yaml` and the `/docs` page, served by the API                          |
| `internal/handlers`   | One handler per area, together implementing `api.StrictServerInterface`          |
| `internal/models`     | Domain types and their rules; imports neither `api` nor `store`                  |
| `internal/store`      | Postgres reads and writes, in terms of `models`                                  |
| `internal/database`   | pgx pool and the migrator; `migrations/NNNN_name.sql`                            |
| `internal/config`     | Env vars → `Config`. The only place that reads the environment                   |
| `internal/middleware` | Request ID, access log, panic recovery, body limit                               |
| `internal/httpx`      | `JSON` and `Error` for responses outside the generated handlers                  |
| `scripts/`            | `with-env.sh`, `reset.sh`, `rename.sh`, `test-env.sh`. Run through make          |

## Go conventions

- **Contract first.** An endpoint starts in `api/openapi.yaml`, then `make generate`. Never
  edit `api/api.gen.go`; change the spec. Request and response shapes come from the generated
  types, never hand-written structs.
- **Standard library first.** Routing is `net/http` (the generated code registers on our
  `ServeMux`) and logging is `log/slog`. Don't add a router, logger, config, or migration
  library. Direct dependencies are `pgx` and `oapi-codegen/runtime`; a new one needs a reason.
- **Layers, one file per feature in each.** Dependencies point one way: `router` → `handlers`
  → `models`, and `store` → `models`. `models` imports neither `api` nor `store`; `handlers`
  never imports `store`. A feature (todo is the template) adds `models/<name>.go`,
  `store/<name>.go`, and `handlers/<name>.go`. The handler declares the store interface it needs
  (`TodoStore`), so tests call its methods directly with an in-memory fake. Wire it by
  embedding the handler in `Handlers` (`handlers/handlers.go`) and constructing it in
  `router.Routes`; the compiler then checks every operation in the spec is implemented. Map
  between `models` and `api` types in the handler.
- **Responses and errors.** Return the generated typed response for every expected outcome:
  400 for malformed input, 404, 422 for well-formed but invalid input. Build error bodies with
  `api.NewError(code, message)`; the shape is always `{"error": {"code", "message"}}`, with codes
  from the spec. Return a Go `error` only for the unexpected: the server logs it and answers a
  generic 500, so database details never reach the client. Wrap errors with context:
  `fmt.Errorf("create todo: %w", err)`.
- **Validation.** Domain rules (trimming, lengths, required fields) live in the handler, and
  limits match the spec (`maxLength: 200` in YAML is enforced in Go). Bad JSON and unparseable
  path values are rejected before the handler, as 400s.
- **Logging.** Use the `*Context` methods inside a request (`log.ErrorContext(ctx, ...)`), so the
  line carries the request ID. Keys are `snake_case`, errors go under `"error"`. Never log
  request bodies or credentials; keys containing password, secret, token, authorization,
  cookie, or api_key are redacted anyway, but don't rely on that. `debug` is for detail you'd
  only want while investigating; `LOG_LEVEL` turns it on without a rebuild.
- **Telemetry.** Spans come from the instrumentation (HTTP, pgx, Temporal); add your own only
  around work that isn't already a request, query, or activity
  (`otel.Tracer("...").Start(ctx, ...)`), and always from the request's `ctx`. New metrics use
  the global OTel meter (`otel.Meter`), with low-cardinality attributes only: never user IDs,
  todo IDs, or raw paths. Metrics are served on the internal metrics port, never the API port.
- **Auth.** Which operations need which role lives in the spec
  (`security: [{ bearerAuth: [role] }]`); `internal/auth` enforces it from the generated
  `api.BearerAuthScopes`. Handlers never read the `Authorization` header: they call
  `auth.PrincipalFrom(ctx)`. Anything a user owns is scoped by `Principal.Subject` in the
  store, and someone else's resource is a 404, never a 403. Keycloak's realm is
  `deploy/keycloak/realm-app.json`; change roles and clients there, not by hand in the console.
- **Workflows.** Writes that should be retried and recorded run as Temporal workflows: the
  handler calls a `...Writer` interface, implemented by a runner in `internal/workflows` that
  starts the workflow through `Runner.Run` and waits. Workflow functions only call activities
  and `workflow.*` APIs (no clock, randomness, or I/O); the I/O lives in `internal/activities`.
  Activities must be safe to retry; mark errors that retrying can't fix as non-retryable.
  Register every workflow and activity in `cmd/worker`. Reads don't go through Temporal.
- **Security headers.** `middleware.SecurityHeaders` covers every API response. A handler that
  serves HTML sets its own Content-Security-Policy, allowing inline scripts by hash only (see
  `internal/docs`). Loosening the web policy in `next.config.ts` needs a stated reason.
- **Context.** Pass the handler's `ctx` to every store call. Never use `context.Background()`
  in a request path.
- **Migrations.** Add a new numbered file; never edit one that has shipped (checksums catch it,
  and the server refuses to start). New files must sort after every applied one. They run
  through `make db-migrate`, never on server start. Forward-only: undo with a new migration.
  Tables with `updated_at` attach the `set_updated_at()` trigger from `0001_init.sql`.
- **Config.** New settings go in `internal/config` with a default that works locally, and in
  `.env.example` with a comment. Invalid values fail startup; nothing fails later at runtime.
  Database settings are the libpq `PG*` variables; don't introduce a second copy of them.
- **JSON.** Field names are camelCase (`createdAt`), defined once, in the spec.

## Removable features

`make reset` deletes the todo demo. Code that exists only for a feature must be removable
without editing anything by hand:

- Files the feature owns are listed in `FEATURE_PATHS` in `scripts/reset.sh`: its file in each
  layer, its migration, its web files, and its e2e test.
- Lines it adds to shared files carry a marker comment: a single line ending in
  `feature:todo`, or a block between a `feature:todo:begin` line and a `feature:todo:end` line.
  This includes the spec (`# feature:todo:begin` in YAML) and table rows in shared tests. In JSX,
  use `{/* ... */}` comments on their own lines.
- Reset regenerates the Go and TypeScript code from the trimmed spec, so schemas a feature adds
  go inside its marked block too.
- New code built on the demo, as opposed to replacing it, must not depend on it.

## Don't

- Commit `.env`, credentials, or build output (`bin/`, `web/.next`, `coverage.out`,
  `web/test-results`). Generated contract code (`api/api.gen.go`, `web/lib/api/schema.ts`) is
  the exception: it is committed, and CI checks it matches the spec.
- Read `.env` from Make, or write a database password in a second place.
- Use `pkill -f` or `grep` patterns that match your own shell command line when stopping dev
  servers. Kill by PID, or match the binary path (`node_modules/.bin/next`).
- Bypass the Makefile in docs or CI. If a command matters, it gets a target.
