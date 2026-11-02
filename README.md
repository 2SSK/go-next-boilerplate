# go-next-boilerplate

A starter for a Go API with a Next.js frontend. It ships with a small todo app that shows the
full round trip (browser → Next.js → Go → Postgres); `make reset` removes it.

- **API:** Go standard library (`net/http`, `slog`) and `pgx`
- **Contract:** OpenAPI (`api/openapi.yaml`); the Go server and TypeScript types are generated
  from it
- **Database:** Postgres 17 with plain SQL migrations
- **Web:** Next.js App Router, shadcn/ui, Tailwind v4

Need auth, background workflows, or monitoring? Use the `platform` branch.

## Quick start

Needs Go 1.25+, Node 22+, and Docker.

```sh
gh repo create my-app --template 2SSK/go-next-boilerplate --private --clone
cd my-app
make setup                                             # .env, Go modules, npm install
make dev                                               # web :3000, API :8080
make rename NAME=my-app MODULE=github.com/you/my-app   # make it yours
make reset                                             # optional: remove the todo demo
```

The API documents itself at http://localhost:8080/docs.

## Commands

`make` lists them all.

| Command                 | What it does                                         |
| ----------------------- | ---------------------------------------------------- |
| `make dev`              | Postgres, migrations, API, and web; Ctrl-C stops all |
| `make generate`         | Regenerate Go and TypeScript code from the spec      |
| `make db-migrate`       | Apply pending migrations                             |
| `make test`             | Go unit tests                                        |
| `make check`            | Lint, contract check, tests, build: what CI runs     |
| `make up` / `make down` | Whole stack in Docker, production builds             |
| `make e2e`              | Browser tests against `make up`                      |
| `make reset`            | Remove the todo demo                                 |

## Layout

```
api/            openapi.yaml and the Go code generated from it
cmd/api/        `api` serves HTTP; `api migrate` applies migrations
internal/
  router/       routes, middleware, the http.Server
  handlers/     one handler per area, implementing the generated interface
  models/       domain types and rules
  store/        Postgres queries
  database/     connection pool, migrator, migrations/*.sql
  config/       environment variables → Config
web/
  app/          pages and globals.css (design tokens)
  config/       site.ts: name, navigation, navbar style
  lib/          typed API client, generated types
docs/adr/       why things are the way they are
```

## How it works

- **Spec first.** Change `api/openapi.yaml`, run `make generate`, and the build tells you what
  to implement. CI fails if generated code is stale.
- **One origin.** The browser only talks to Next.js; it forwards `/api/*` to the Go API, so
  there's no CORS.
- **Errors** always look like `{"error": {"code": "not_found", "message": "..."}}`.
- **Health:** `/livez` (process up) and `/readyz` (database reachable).
- **Migrations** run as a deploy step (`make db-migrate`), never on server start. The server
  refuses to run against a pending or edited migration.

## Adding a feature

Copy the todo demo, one file per layer. For `notes`:

1. Add the endpoints to `api/openapi.yaml`, then `make generate`.
2. Add `internal/database/migrations/0003_notes.sql`, then `make db-migrate`.
3. Add `models/note.go`, `store/notes.go`, and `handlers/notes.go`.
4. Wire the handler in `handlers/handlers.go` and `router/router.go`.
5. Add `web/app/notes/` and a nav entry in `web/config/site.ts`.

## Configuration

Environment variables, documented in `.env.example` (`make setup` copies it to `.env`). The
ones you'll touch most:

| Variable                             | Default                                              |
| ------------------------------------ | ---------------------------------------------------- |
| `APP_ENV`                            | `development` (text logs) or `production`            |
| `PORT`, `WEB_PORT`                   | `8080`, `3000`                                       |
| `PGHOST`, `PGPORT`                   | `localhost`, `5432`                                  |
| `PGUSER`, `PGPASSWORD`, `PGDATABASE` | `app`                                                |
| `API_URL`                            | `http://localhost:8080`, where Next.js finds the API |

Quote values with spaces, `#`, or `$` in single quotes. A variable set in your shell overrides
`.env`.

## Reset

`make reset` deletes the files the todo demo owns and the lines it added to shared files
(marked `feature:todo`), then regenerates code. You're left with `GET /`, health checks, docs,
and the landing page. CI checks this on every push.

## Deploying

Two images: `Dockerfile` (the API, a static binary running as non-root) and `web/Dockerfile`
(Next.js standalone). Run `api migrate` before starting a new API version. `make up` runs both
locally to test production builds.

## License

MIT
