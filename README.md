# go-next-boilerplate

A minimal full-stack starter for Go on the backend and Next.js on the frontend. Clone it, run
two commands, and start building. It ships with a small todo app that shows the full round trip
(browser → Next.js → Go → Postgres); `make reset` strips it out when you want a blank slate.

| Layer    | Choice                                                                            |
| -------- | --------------------------------------------------------------------------------- |
| Contract | OpenAPI 3 (`api/openapi.yaml`): Go server and TypeScript types generated from it  |
| API      | Go standard library `net/http`, `log/slog`, `pgx` for Postgres                    |
| Database | Postgres 17, plain SQL migrations with checksums, run as a deploy step            |
| Web      | Next.js (App Router), shadcn/ui, Tailwind v4, Inter + JetBrains Mono              |
| Tooling  | Make, Docker Compose, golangci-lint, ESLint, Prettier, Playwright, GitHub Actions |

The Go API has two dependencies: `pgx` and `oapi-codegen/runtime` (used by the generated code).

## Quick start

Needs Go 1.25+, Node 22+, and Docker.

```sh
gh repo create my-app --template 2SSK/go-next-boilerplate --private --clone
cd my-app
make setup   # .env, Go modules, npm install
make dev     # Postgres, Keycloak, Temporal, migrations, API on :8080, worker, web on :3000
```

Open http://localhost:3000. The status pill on the landing page turns green once the API and
database answer. The API serves its own docs at http://localhost:8080/docs.

Then make it yours:

```sh
make rename NAME=my-app MODULE=github.com/you/my-app   # module path, app name, compose project
make reset                                             # optional: remove the todo demo
```

## Commands

Run `make` to list them all. The ones you'll use most:

| Command                 | What it does                                                                            |
| ----------------------- | --------------------------------------------------------------------------------------- |
| `make dev`              | Postgres, Keycloak, Temporal, migrations, API, worker, and web; Ctrl-C stops everything |
| `make tp-up`            | Temporal dev server alone (UI at http://localhost:8233)                                 |
| `make generate`         | Regenerate the Go server and TypeScript types from the spec                             |
| `make db-migrate`       | Apply pending migrations                                                                |
| `make db-status`        | List applied and pending migrations                                                     |
| `make test`             | Go unit tests (race detector) and the `.env` round-trip test                            |
| `make test-integration` | Go tests against the real database                                                      |
| `make lint`             | gofmt, go vet, golangci-lint, ESLint, tsc, Prettier                                     |
| `make check`            | lint + contract check + test + build, the gates CI runs                                 |
| `make up` / `make down` | the whole stack in Docker, production builds                                            |
| `make e2e`              | Playwright browser tests against the running stack (`make up` first)                    |
| `make db-shell`         | psql into the dev database                                                              |
| `make db-reset`         | wipe the dev database                                                                   |
| `make reset`            | remove the todo demo (asks first; `YES=1` skips the prompt)                             |

## Layout

```
api/                  the contract: openapi.yaml, and the Go code generated from it
cmd/api/              entry point: `api` serves HTTP, `api migrate` applies migrations
cmd/worker/           runs Temporal workflows and activities
internal/
  router/             mounts the generated routes, the docs, the JSON 404; middleware; http.Server
  docs/               /openapi.yaml and the interactive /docs page, served by the API
  handlers/           implement the API operations, one handler per area (system, todo)
  models/             domain types and rules (Todo, NormalizeTitle, ErrNotFound)
  store/              Postgres reads and writes, returning models
  database/           pgx pool and the migrator (migrations/*.sql)
  workflows/          Temporal workflows, and the Runner the API starts them with
  activities/         the steps workflows run (database writes), retried by Temporal
  temporal/           Temporal client setup
  config/             env vars → Config, with defaults and validation
  middleware/         request ID, access log, panic recovery, body limit
  httpx/              JSON responses outside the generated handlers
web/
  app/                pages (landing, todos) and globals.css (the design tokens)
  components/         navbar, theme toggle, API status, ui/ (shadcn)
  config/site.ts      app name, description, navbar style, and navigation
  lib/api.ts          typed client for the browser; lib/api-server.ts for server components
  lib/api/schema.ts   TypeScript types generated from the spec
  e2e/                Playwright tests
scripts/              with-env, reset, rename, and the .env test; run through make
docs/                 ROADMAP.md and the architecture decision records (adr/)
```

## How the pieces connect

**The contract comes first.** `api/openapi.yaml` describes every endpoint. `make generate`
produces the Go server interface and types (`api/api.gen.go`) and the TypeScript types
(`web/lib/api/schema.ts`). Handlers implement the generated interface, and the web app calls the
API through a client typed by the same spec, so a change on one side that the other doesn't
match fails to compile. CI fails if the committed generated code is out of date.

**The browser only talks to Next.js for the app.** `next.config.ts` forwards `/api/*`, `/readyz`
and `/livez` to `API_URL`, so there is no CORS to configure. Server components call the API
directly.

**The API documents itself.** It serves the contract at `/openapi.yaml` and an interactive
reference at `/docs` (Scalar, loaded from a CDN, with its telemetry and hosted AI turned off).
These are not proxied through Next.js, so they are reachable exactly where the API is: keep the
API port internal in production and the docs stay internal too, or set `DOCS_ENABLED=false`.

**Security headers.** Every API response carries `nosniff`, `DENY` framing, `no-referrer`, and a
Content-Security-Policy that blocks everything (it's JSON); the docs page sets a strict policy
of its own that allows only Scalar's script and its init script, by hash. Every web page gets a
static Content-Security-Policy and the usual hardening headers from `next.config.ts`. The web
policy allows inline scripts: the strict alternative, nonces, would force every page to render
per request and discard the static shells. HTTPS and HSTS belong to whatever terminates TLS.

**Errors have one shape:** `{"error": {"code": "not_found", "message": "..."}}`. Branch on `code`;
`message` is safe to show people.

**Health:** `/livez` says the process is up and never checks dependencies (for restarts).
`/readyz` also pings the database and answers 503 when it can't (for load balancers).

## Configuration

Environment variables only. `make setup` copies `.env.example` to `.env`, where each one is
explained. Make never parses `.env` itself; targets load it through `scripts/with-env.sh`, and
Docker Compose reads it directly.

**One rule:** wrap a value in single quotes if it contains spaces, `#`, `$`, or quotes, for
example `PGPASSWORD='p@ss $word #1'`. Both readers then see exactly the same value, and
`make test` proves it with exactly such a password. A variable already set in your shell wins
over `.env` for both readers, so `PGPORT=55432 make dev` works as expected.

| Variable                                              | Default                                               | Used by      |
| ----------------------------------------------------- | ----------------------------------------------------- | ------------ |
| `APP_ENV`                                             | `development` (text logs) or `production` (JSON)      | API          |
| `LOG_LEVEL`                                           | `info` in production, `debug` otherwise               | API          |
| `DOCS_ENABLED`                                        | `true`; `false` turns off `/docs` and `/openapi.yaml` | API          |
| `PORT`                                                | `8080`                                                | API          |
| `SHUTDOWN_TIMEOUT`                                    | `10s`, the drain time after SIGTERM                   | API          |
| `PGHOST`, `PGPORT`                                    | `localhost`, `5432` (also the host port in Compose)   | API, Compose |
| `PGUSER`, `PGPASSWORD`, `PGDATABASE`                  | `app` / `app` / `app`                                 | API, Compose |
| `PGSSLMODE`                                           | `disable`                                             | API          |
| `DATABASE_URL`                                        | unset; when set, it wins over the `PG*` values        | API          |
| `KEYCLOAK_PORT`                                       | `8180`                                                | Compose      |
| `OIDC_ISSUER`, `OIDC_AUDIENCE`                        | `http://localhost:8180/realms/app`, `api`             | API          |
| `OIDC_DISCOVERY_URL`                                  | unset; Keycloak's internal address in Docker          | API          |
| `TEMPORAL_ADDRESS`                                    | `localhost:7233`                                      | API, worker  |
| `TEMPORAL_NAMESPACE`, `TEMPORAL_TASK_QUEUE`           | `default`, `app`                                      | API, worker  |
| `TEMPORAL_PORT`, `TEMPORAL_UI_PORT`                   | `7233`, `8233`                                        | Compose      |
| `APP_URL`                                             | `http://localhost:3000`, this app's public URL        | Web          |
| `AUTH_ISSUER`, `AUTH_CLIENT_ID`, `AUTH_CLIENT_SECRET` | the dev realm's values                                | Web          |
| `AUTH_SECRET`                                         | required in production: 32+ random characters         | Web          |
| `AUTH_INTERNAL_ISSUER`                                | unset; Keycloak's internal address in Docker          | Web          |
| `API_URL`                                             | `http://localhost:8080`                               | Web          |
| `WEB_PORT`                                            | `3000`                                                | Web          |

The database credentials are the standard libpq variables, written once: the API builds its
connection from them (escaping any character correctly), Compose creates the database with them,
and psql reads them as-is. If port 5432 is taken, change `PGPORT` and nothing else.

## Authentication (platform)

Users, passwords, and roles live in Keycloak (`make kc-up`, console at
http://localhost:8180, `admin`/`admin`). Dev users: `alice`/`alice` (role `user`) and
`admin`/`admin` (`user` and `admin`). New users get `user` automatically.

- **Sign-in happens on the app's own page** (`/login`). The Next.js server exchanges the
  username and password with Keycloak and keeps the tokens in an encrypted, `httpOnly`
  cookie; browser JavaScript never sees a token. `proxy.ts` refreshes tokens before they
  expire and sends signed-out visitors of protected pages to `/login`.
- **The browser calls the API through `/api/*`**, a Next.js route that attaches the token
  on the server and refuses cross-site writes. Server components use `serverApi()`.
- **The API checks every token** (signature, issuer, expiry, and that it was issued for
  this API) and enforces the roles each operation declares in the spec:
  `security: [{ bearerAuth: [user] }]`. Missing or bad token: 401; missing role: 403.
- **Todos are per user.** Another user's todo is a 404, so its existence isn't revealed.
- **To protect a page**, add its path to `protectedPaths` in `web/config/site.ts`.

The tradeoff of an in-app sign-in form (no Keycloak MFA, password reset, or social login)
and how to switch back to Keycloak's own pages are in `docs/adr/0005`.

## Workflows (platform)

Todo writes run as Temporal workflows; reads go straight to Postgres.

```
browser → web → API ──start workflow──▶ Temporal ◀──poll── worker ──activity──▶ Postgres
                 └──── waits for the result (up to 10s) ◀──────────┘
```

- **The API starts the workflow and waits**, so responses are the same as before (201, 404,
  ...). If Temporal is down, or no worker finishes in time, writes answer **503** while reads
  keep working.
- **The worker** (`make dev-worker`, or the `worker` service in Docker) does the work. Failed
  activities are retried with backoff, up to 5 attempts; a missing todo isn't retried.
- **Retries are safe.** The workflow ID includes the user and the `X-Request-ID`, so the same
  request sent twice is done once.
- **See every write** in the Temporal UI at http://localhost:8233: inputs, attempts, errors.

To run a new write through Temporal: add an activity method in `internal/activities`, a
workflow in `internal/workflows`, a small runner like `TodoRunner`, and register both in
`cmd/worker`. Details and tradeoffs are in `docs/adr/0006`.

## Migrations

Plain SQL files in `internal/database/migrations/`, applied in name order. Schema changes are a
deploy step, never a side effect of starting the server:

- `make db-migrate` (or `api migrate`) applies pending files, each in its own transaction, under
  an advisory lock so concurrent runs can't collide.
- The server checks the schema at startup and refuses to run if a migration is pending or an
  applied file was edited (checksums are recorded).
- In Docker, a one-shot `migrate` service runs before the API starts.
- Migrations are forward-only: to undo a change, write a new migration. See
  `docs/adr/0002-forward-only-migrations.md`.

## Adding a feature

The todo demo is the template; each layer has one file for it. A new resource, say `notes`:

1. **Contract.** Add the paths and schemas to `api/openapi.yaml`, then `make generate`. The Go
   build now fails until the new operations are implemented, which is the point.
2. **Migration.** Add `internal/database/migrations/0003_notes.sql`, then `make db-migrate`.
3. **Model.** `internal/models/note.go`: the domain type and its rules (like `NormalizeTitle`).
4. **Store.** `internal/store/notes.go`: the Postgres reads and writes, returning models.
5. **Handler.** `internal/handlers/notes.go`: a `NotesStore` interface and a `NotesHandler`
   whose methods implement the generated operations. Tests call those methods directly with an
   in-memory store, so they need no database.
6. **Wire.** Embed `*NotesHandler` in `Handlers` (`internal/handlers/handlers.go`) and construct
   it in `router.Routes`. If an operation is missing, the build says which.
7. **Web.** Add `web/lib/notes.ts` (types from `lib/api/schema.ts`, calls through `api`), a page
   in `web/app/notes/`, and a nav entry in `web/config/site.ts`. For more shadcn components, run
   `npx shadcn@latest add <name>` in `web/`.

## How reset works

The demo owns some files outright (`todo.go` in models, store and handlers, its migration,
`web/app/todos/`, its e2e test, and so on) and adds a few lines of wiring to shared files, the spec included. Those lines
are tagged with a comment: either on the same line, or between a begin and an end comment
(`feature:todo:begin` and its matching `:end`). `scripts/reset.sh` deletes the owned files and
the tagged lines, regenerates the Go and TypeScript code from the trimmed spec, then runs gofmt,
`go mod tidy`, the build, and Prettier.

After a reset you have `GET /`, `/livez`, `/readyz`, `/docs`, and the landing page. The `todos`
table stays in your local database until `make db-reset`. CI runs a reset on every push and
checks the result, so this path can't silently break.

## Design

The web app follows one design language: minimal, one accent colour, soft raised surfaces, and
three motion speeds on a single easing curve. Light mode is neutral paper with Tokyo Night Day
accents; dark mode is Tokyo Night. Every token is a CSS variable in `web/app/globals.css`, and the
shadcn components read the same variables, so changing a colour there restyles everything.

The navbar comes in two styles, set by `navbar` in `web/config/site.ts`. `"bar"` (the default) is
a full-width sticky bar for platform and infra tools. `"island"` is a floating capsule for
products and landing pages. Page spacing follows the choice automatically.

The full design system (tokens, type scale, motion, component rules) and the rendering and
performance rules are in `web/AGENTS.md`.

## Deploying

`make up` runs the production images locally; it is a production-like test environment, not a
deployment (no TLS, no backups, one replica). The API image (`Dockerfile`) is a static binary on
Alpine, runs as a non-root user, and has a health check on `/livez`. Run `api migrate` as a
release step (a Kubernetes Job, a Compose one-shot, a platform release command) before starting
new API versions. The web image (`web/Dockerfile`) is Next.js standalone output; `API_URL` is
used both at build time (rewrites) and at runtime (server components), so set it in both.

## License

MIT
