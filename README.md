# go-next-boilerplate · platform

A starter for a Go API with a Next.js frontend, plus sign-in, background workflows, and
monitoring. It ships with a small todo app that runs through every piece; `make reset` removes
it. For the plain starter without these, use the `main` branch.

- **API:** Go standard library (`net/http`, `slog`) and `pgx`
- **Contract:** OpenAPI (`api/openapi.yaml`); the Go server and TypeScript types are generated
  from it
- **Database:** Postgres 17 with plain SQL migrations
- **Web:** Next.js App Router, shadcn/ui, Tailwind v4
- **Auth:** Keycloak, with sign-in on the app's own page
- **Workflows:** Temporal, with a separate worker
- **Monitoring:** OpenTelemetry, Prometheus, Grafana, Tempo, Loki

## Quick start

Needs Go 1.26+, Node 22+, and Docker.

```sh
gh repo create my-app --template 2SSK/go-next-boilerplate --private --clone
cd my-app && git checkout platform
make setup                                             # .env, Go modules, npm install
make dev                                               # web :3000, API :8080, worker
make mon-up                                            # optional: Grafana :3001
make rename NAME=my-app MODULE=github.com/you/my-app   # make it yours
make reset                                             # optional: remove the todo demo
```

Sign in at http://localhost:3000 as `alice`/`alice`. The API documents itself at
http://localhost:8080/docs.

## Commands

`make` lists them all.

| Command                    | What it does                                                   |
| -------------------------- | -------------------------------------------------------------- |
| `make dev`                 | Postgres, Keycloak, Temporal, migrations, API, worker, and web |
| `make mon-up` / `mon-down` | Grafana :3001, Prometheus :9090, Temporal UI :8233             |
| `make generate`            | Regenerate Go and TypeScript code from the spec                |
| `make db-migrate`          | Apply pending migrations                                       |
| `make test`                | Go unit tests                                                  |
| `make check`               | Lint, contract check, tests, build: what CI runs               |
| `make up` / `make down`    | Whole stack in Docker, production builds                       |
| `make e2e`                 | Browser tests against `make up`                                |
| `make reset`               | Remove the todo demo                                           |

## Layout

```
api/              openapi.yaml and the Go code generated from it
cmd/api/          `api` serves HTTP; `api migrate` applies migrations
cmd/worker/       runs Temporal workflows and activities
internal/
  router/         routes, middleware, the http.Server
  handlers/       one handler per area, implementing the generated interface
  models/         domain types and rules
  store/          Postgres queries
  workflows/      Temporal workflows, and the runner the API starts them with
  activities/     the work workflows do (database writes), retried by Temporal
  auth/           access-token checks and roles
  telemetry/      tracing, log export, /metrics
  database/       connection pool, migrator, migrations/*.sql
  config/         environment variables → Config
web/
  app/            pages, sign-in, and the /api proxy that attaches the token
  config/         site.ts: name, navigation, protected pages
  lib/            typed API client, session, generated types
deploy/           Keycloak realm, monitoring configs, Grafana dashboard
docs/adr/         why things are the way they are
```

## How it works

```
browser → Next.js → API ──start workflow──▶ Temporal ◀── worker ──▶ Postgres
                     └── reads ─────────────────────────────────────▶ Postgres
```

- **Spec first.** Change `api/openapi.yaml`, run `make generate`, and the build tells you what
  to implement.
- **Sign-in.** The Next.js server signs users in with Keycloak and keeps the tokens in an
  encrypted cookie; the browser never sees a token. The API checks every token and the roles
  each operation declares in the spec. Todos are per user.
- **Writes run as Temporal workflows**, executed by the worker with retries. The API waits for
  the result, so responses are unchanged. If Temporal is down, writes return 503 and reads keep
  working. Sending the same `X-Request-ID` twice does the write once.
- **One trace per request**, from the Next.js route through the API, Temporal, the worker, and
  each SQL query. Every log line carries its `trace_id`.
- **Errors** always look like `{"error": {"code": "not_found", "message": "..."}}`.
- **Health:** `/livez` (process up) and `/readyz` (database reachable).
- **Migrations** run as a deploy step (`make db-migrate`), never on server start.

## Monitoring

`make mon-up`, then open Grafana at http://localhost:3001:

- **App overview** dashboard: request rate, errors, and latency per route; workflow and activity
  rates and failures; database pool; Go runtime.
- **Explore → Tempo:** traces. **Explore → Loki:** logs, filter with `| trace_id="..."`.
  Traces, logs, and latency graphs link to each other.
- **Temporal UI** at http://localhost:8233: every workflow with its inputs, attempts, and errors.

Metrics are served on internal ports (API 9464, worker 9465), never the public API port. To
send traces and logs elsewhere, change `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Adding a feature

Copy the todo demo, one file per layer. For `notes`:

1. Add the endpoints to `api/openapi.yaml` (with `security: [{ bearerAuth: [user] }]`), then
   `make generate`.
2. Add `internal/database/migrations/0004_notes.sql`, then `make db-migrate`.
3. Add `models/note.go`, `store/notes.go`, and `handlers/notes.go`.
4. For writes that should be retried and recorded: an activity in `activities/`, a workflow in
   `workflows/`, and register both in `cmd/worker`.
5. Wire the handler in `handlers/handlers.go` and `router/router.go`.
6. Add `web/app/notes/`, a nav entry, and the path in `protectedPaths` (`web/config/site.ts`).

## Configuration

Environment variables, documented in `.env.example` (`make setup` copies it to `.env`). The
ones you'll touch most:

| Variable                             | Default                                       |
| ------------------------------------ | --------------------------------------------- |
| `APP_ENV`                            | `development` (text logs) or `production`     |
| `PORT`, `WEB_PORT`                   | `8080`, `3000`                                |
| `PGHOST`, `PGPORT`                   | `localhost`, `5432`                           |
| `PGUSER`, `PGPASSWORD`, `PGDATABASE` | `app`                                         |
| `OIDC_ISSUER`, `AUTH_ISSUER`         | the dev Keycloak realm                        |
| `AUTH_SECRET`                        | required in production: 32+ random characters |
| `TEMPORAL_ADDRESS`                   | `localhost:7233`                              |
| `OTEL_EXPORTER_OTLP_ENDPOINT`        | the local collector; empty exports nothing    |

Quote values with spaces, `#`, or `$` in single quotes. A variable set in your shell overrides
`.env`.

## Reset

`make reset` deletes the files the todo demo owns and the lines it added to shared files
(marked `feature:todo`), then regenerates code. Sign-in, the worker, and monitoring stay. CI
checks this on every push.

## Deploying

One image runs both the API and the worker (`Dockerfile`); the web app has its own
(`web/Dockerfile`). Run `api migrate` before starting a new API version. `make up` runs
everything locally to test production builds.

The Compose setup is for development. In production you need a real Temporal cluster (or
Temporal Cloud), Keycloak with its own database, HTTPS, and real secrets; the dev users and the
dev client secret in `deploy/keycloak/realm-app.json` must go.

## License

MIT
