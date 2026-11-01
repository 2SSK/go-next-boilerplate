# 4. Layered internal packages: router, handlers, models, store

Status: accepted (2026-11-01)

## Context

The first version grouped code by feature (`internal/todo` held the model, store, and handler).
That makes deleting a feature a one-folder change, but the author prefers layers: a central
router, central models, central handlers. It's familiar, it's what most Go services look like,
and it makes "where does X live" answerable without knowing the feature.

## Decision

- `internal/router`: mounts every generated route, the spec, `/docs`, the JSON 404, and the
  middleware chain; builds the `http.Server`. The route table itself is `api/openapi.yaml`.
- `internal/handlers`: one handler type per area, embedded into `Handlers`, which must satisfy
  `api.StrictServerInterface` (a compile-time check). Each handler declares the storage
  interface it needs.
- `internal/models`: domain types and rules. No imports of `api` or `store`.
- `internal/store`: Postgres implementations, in terms of `models`.
- A feature is one file per layer, plus its migration and web files. Dependencies point one way:
  router → handlers → models, store → models.
- No service layer until logic outgrows a handler; then add `internal/services` with the same
  one-file-per-feature rule.

Wire models stay generated in `api/`; `models` holds the domain types only.

## Consequences

- Adding a feature touches four packages instead of one; `make reset` lists the feature's file in
  each layer rather than one folder.
- A shared models package can turn into a catch-all; keep it to domain types and their rules, and
  keep each feature in its own file.
