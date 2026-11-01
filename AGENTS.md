- Do not preserve backward compatibility. Remove obsolete paths instead of adding compatibility layers, fallbacks, or migrations.
- Choose the simplest implementation that fully meets the current requirements. Avoid speculative abstractions, configurations, and indirections.
- Grow the system in layers. Start from the smallest version that works to end, and add each new capability on top of a product that already works. Never trade a working product for unfinished complexity.
- Keep components modular and concerns clearly separated.
- Prefer established, well-maintained libraries when they reduce overall complexity or improve reliaability. Do not reimplement common functionality without a clear reason.
- Lean on the dependencies already in the project before writing your own implementation or adding packages. Do not assume a library lacks a capability without checking its documentation and types.
- Make architectural decisions for the long term. Do not acccept a stopgap that only works for now and is meant to be replaced later.

# Agent guide

A Go API (`cmd/`, `internal/`) and a Next.js frontend (`web/`) in one repo. The browser only
talks to Next.js, which forwards `/api/*` and `/healthz` to the API. Web-specific rules and the
design system are in `web/AGENTS.md`; read it before touching anything under `web/`.
Its "Rendering and performance" rules are not optional: shared chrome (navbar, footer) lives
in layouts, never pages; client components stay small leaves; page data loads on the server
and streams behind `<Suspense>`.

## Commands

Run everything through the Makefile; `make` lists the targets.

- `make dev`: Postgres + API (:8080) + web (:3000). Needs `make setup` once.
- `make test`: Go unit tests, with the race detector.
- `make test-integration`: Go tests against Postgres (`make db-up` first).
- `make lint`: gofmt, go vet, golangci-lint, ESLint, tsc, Prettier.
- `make check`: lint + test + build. CI runs the same gates.
- `make fmt`: format Go and web code.

## Definition of done

- `make check` passes with no warnings. ESLint runs with `--max-warnings=0`.
- New behavior has tests: table-driven Go tests with subtests, one per case.
- If you changed anything a reset touches, `make reset` still produces a building tree. CI
  checks this too.
- The README is updated when commands, env vars, or endpoints change. Every env var is also
  documented in `.env.example`.

## Layout

| Path                  | Holds                                                                      |
| --------------------- | -------------------------------------------------------------------------- |
| `cmd/api`             | Startup only: config, database, server, graceful shutdown                  |
| `internal/config`     | Env vars → `Config`. The only place that reads the environment             |
| `internal/database`   | pgx pool and the migrator; `migrations/NNNN_name.sql`                      |
| `internal/httpx`      | `JSON`, `Error`, `Decode`. Every handler responds through these            |
| `internal/middleware` | Request ID, access log, panic recovery                                     |
| `internal/server`     | Route table (`Routes`), `GET /`, `GET /healthz`                            |
| `internal/<feature>`  | One package per resource: model, `Store`, Postgres store, `Handler`, tests |
| `scripts/`            | `reset.sh`, `rename.sh`. Run through make, never by hand                   |

## Go conventions

- **Standard library first.** Routing is `net/http` with method patterns (`"GET /items/{id}"`)
  and logging is `log/slog`. Don't add a router, logger, config, or migration library;
  `pgx` is the only dependency, and a new one needs a reason.
- **Feature packages, not layers.** A resource lives in one package (`internal/todo` is the
  template). Its handler depends on a small `Store` interface, so tests use an in-memory fake.
  Wire it in `internal/server/server.go` with `Register(mux, "/api/v1/<name>")`.
- **Errors.** Wrap with context: `fmt.Errorf("create todo: %w", err)`. Store-level sentinel
  errors (`ErrNotFound`) map to status codes in the handler. A 500 logs the real error and
  returns `"internal server error"`, so database details never reach the client.
- **Requests.** Decode with `httpx.Decode`, which rejects unknown fields and trailing data.
  Validate in the handler: 400 for malformed input, 422 for well-formed but invalid input.
  Responses are JSON; errors are always `{"error": "..."}`.
- **Context.** Pass `r.Context()` to every store call. Never use `context.Background()` in a
  request path.
- **Migrations.** Add a new numbered file; never edit one that has shipped. Each file runs
  once, in a transaction, on API startup. Tables with `updated_at` attach the `set_updated_at()`
  trigger from `0001_init.sql`.
- **Config.** New settings go in `internal/config` with a default that works locally, and in
  `.env.example` with a comment. Invalid values fail startup; nothing fails later at runtime.
- **JSON.** Field names are camelCase (`createdAt`); TypeScript types in `web/lib/` mirror
  them exactly.

## Removable features

`make reset` deletes the todo demo. Code that exists only for a feature must be removable
without editing anything by hand:

- Files the feature owns are listed in `FEATURE_PATHS` in `scripts/reset.sh`.
- Lines it adds to shared files carry a marker comment: a single line ending in
  `feature:todo`, or a block between a `feature:todo:begin` line and a `feature:todo:end` line.
  In JSX, use `{/* ... */}` comments on their own lines.
- New code built on the demo, as opposed to replacing it, must not depend on it.

## Don't

- Commit `.env`, credentials, or generated output (`bin/`, `web/.next`, `coverage.out`).
- Use `pkill -f` or `grep` patterns that match your own shell command line when stopping dev
  servers. Kill by PID, or match the binary path (`node_modules/.bin/next`).
- Bypass the Makefile in docs or CI. If a command matters, it gets a target.
