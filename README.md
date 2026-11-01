# go-next-boilerplate

A Go API and a Next.js frontend, wired together. Clone it, run two commands, and start
building. It ships with a small todo app that shows a full round trip (Next.js → Go →
Postgres); `make reset` strips that out when you want a blank slate.

| Layer    | Choice                                                                         |
| -------- | ------------------------------------------------------------------------------ |
| API      | Go standard library `net/http` (1.22+ routing), `log/slog`, `pgx` for Postgres |
| Database | Postgres 17 in Docker, plain `.sql` migrations embedded in the binary          |
| Web      | Next.js (App Router), shadcn/ui, Tailwind v4, Inter + JetBrains Mono           |
| Tooling  | Make, Docker Compose, golangci-lint, ESLint, Prettier, GitHub Actions          |

The only Go dependency is `pgx`.

## Quick start

Needs Go 1.25+, Node 22+, and Docker.

```sh
gh repo create my-app --template 2SSK/go-next-boilerplate --private --clone
cd my-app
make setup   # .env, Go modules, npm install
make dev     # Postgres + API on :8080 + web on :3000
```

Open http://localhost:3000. The status pill on the landing page turns green once the API and
database answer.

Then make it yours:

```sh
make rename NAME=my-app MODULE=github.com/you/my-app   # module path, app name, compose project
make reset                                             # optional: remove the todo demo
```

## Commands

Run `make` to list them all. The ones you'll use most:

| Command                 | What it does                                                |
| ----------------------- | ----------------------------------------------------------- |
| `make dev`              | Postgres, API, and web together; Ctrl-C stops everything    |
| `make test`             | Go unit tests with the race detector                        |
| `make test-integration` | Go tests against the real database                          |
| `make lint`             | gofmt, go vet, golangci-lint, ESLint, tsc, Prettier         |
| `make check`            | lint + test + build, the same gates CI runs                 |
| `make up` / `make down` | the whole stack in Docker, production builds                |
| `make db-shell`         | psql into the dev database                                  |
| `make db-reset`         | wipe the dev database                                       |
| `make reset`            | remove the todo demo (asks first; `YES=1` skips the prompt) |

## Layout

```
cmd/api/              entry point: config, database, server, graceful shutdown
internal/
  config/             env vars → Config, with defaults and validation
  database/           pgx pool + migrator (migrations/*.sql, applied on startup)
  httpx/              JSON response and request-decoding helpers
  middleware/         request ID, access log, panic recovery
  server/             route table, GET / and GET /healthz
  todo/               the demo feature: model, Postgres store, handlers, tests
web/
  app/                pages (landing, todos) and globals.css (the design tokens)
  components/         navbar, theme toggle, API status, ui/ (shadcn)
  config/site.ts      app name, description, navbar style, and navigation
  lib/api.ts          fetch helper for the Go API
scripts/              reset.sh and rename.sh, run through make
```

## How the pieces connect

The browser only talks to Next.js. `next.config.ts` rewrites `/api/*` and `/healthz` to
`API_URL`, so there is no CORS to configure and the API address lives in one env var.

Configuration is environment variables only. `make setup` copies `.env.example` to `.env`, and
the Makefile exports it to every command.

| Variable                                            | Default                                          | Used by |
| --------------------------------------------------- | ------------------------------------------------ | ------- |
| `APP_ENV`                                           | `development` (text logs) or `production` (JSON) | API     |
| `PORT`                                              | `8080`                                           | API     |
| `DATABASE_URL`                                      | `postgres://app:app@localhost:5432/app?...`      | API     |
| `SHUTDOWN_TIMEOUT`                                  | `10s`, the drain time after SIGTERM              | API     |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | `app` / `app` / `app`                            | Compose |
| `DB_PORT`                                           | `5432`, the host port for compose Postgres       | Compose |
| `API_URL`                                           | `http://localhost:8080`, read at build time      | Web     |
| `WEB_PORT`                                          | `3000`                                           | Web     |

`.env.example` explains each one. If port 5432 is already taken, change `DB_PORT` and the port in `DATABASE_URL` together.

## Adding a feature

`internal/todo` is the template. A new resource, say `notes`:

1. **Migration.** Add `internal/database/migrations/0003_notes.sql`. It runs on the next API
   start. Files apply in name order, once each, inside a transaction.
2. **Package.** Copy `internal/todo` to `internal/notes`: a `Store` interface, a Postgres
   implementation, and a `Handler` with a `Register(mux, prefix)` method. Handler tests use an
   in-memory store, so they need no database.
3. **Route.** Add one line in `internal/server/server.go`:
   `notes.NewHandler(notes.NewPostgresStore(d.DB)).Register(mux, "/api/v1/notes")`.
4. **Web.** Add `web/lib/notes.ts` (typed calls through `api()`), a page in `web/app/notes/`,
   and a nav entry in `web/config/site.ts`. For more shadcn components, run
   `npx shadcn@latest add <name>` in `web/`.

## How reset works

The demo owns some files outright (`internal/todo/`, its migration, `web/app/todos/`, and so
on), and adds a few lines of wiring to shared files. Those lines are tagged with a comment:
either on the same line, or between a begin and an end comment (`feature:todo:begin` and its
matching `:end`). `scripts/reset.sh` deletes the owned files, deletes the tagged lines, then
runs gofmt, `go mod tidy`, the build, and Prettier.

After a reset you have `GET /`, `GET /healthz`, and the landing page. The `todos` table stays in
your local database until `make db-reset`. CI runs a reset on every push and builds the result,
so this path can't silently break.

## Design

The web app follows one design language: minimal, one accent colour, soft raised surfaces, and
three motion speeds on a single easing curve. Light mode is neutral paper with Tokyo Night Day
accents; dark mode is Tokyo Night. Every token is a CSS variable in `web/app/globals.css`, and the
shadcn components read the same variables, so changing a colour there restyles everything.

The navbar comes in two styles, set by `navbar` in `web/config/site.ts`. `"bar"` (the default) is
a full-width sticky bar for platform and infra tools. `"island"` is a floating capsule for
products and landing pages. Page spacing follows the choice automatically.

The full design system (tokens, type scale, motion, component rules) is in `web/AGENTS.md`.

Helper classes: `container-page` (page column), `page-top` (space below the navbar), `surface` (raised card), `label` (small caps
heading), `meta` (mono data text), `interactive` (shared transition), `animate-enter`
(page entrance on a fresh load; stagger it with `animationDelay`), and `animate-rise` (something
appearing inside a page). After a client navigation the whole page rises in at once, through a
plain CSS animation that never delays the router.

## Deploying

`make up` runs the production images locally. The API image (`Dockerfile`) is a static binary on
Alpine, runs as a non-root user, and has a health check on `/healthz`. The web image
(`web/Dockerfile`) is Next.js standalone output. Set `APP_ENV=production` and `DATABASE_URL` for
the API. `API_URL` is read when the web image is **built**, because Next.js resolves rewrites
at build time, so pass it as a build arg.

## License

MIT
