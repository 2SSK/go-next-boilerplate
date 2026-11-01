# 1. OpenAPI, spec-first, in the core

Status: accepted (2026-11-01)

## Context

The web app's types were hand-copied from the Go structs (`web/lib/todos.ts`). Nothing caught a
change on one side that the other didn't follow, and the drift grows with every endpoint. The
author's standards also require API documentation for every project, not only large ones.

## Decision

`api/openapi.yaml` is the contract and the source of truth, in the core starter.

- Go: oapi-codegen generates a standard-library (`net/http`) strict server interface and types
  (`api/api.gen.go`). Handlers implement that interface; the router, middleware, and logging
  stay ours. The generator is pinned with Go's `tool` directive and is not in the binary.
- TypeScript: openapi-typescript generates types (`web/lib/api/schema.ts`); openapi-fetch is the
  typed client.
- Generated code is committed. CI regenerates it and fails on any difference.
- Domain validation stays in Go handlers. Full request validation against the spec would need
  kin-openapi, a heavy dependency; the spec documents the limits and the handlers enforce them.
- Generation keeps unused schemas (`skip-prune`), so the shared `Error` schema exists even when no
  feature references it.

Code-first options were rejected: comment-annotated specs (swaggo) drift from the code, and
frameworks that derive the spec (huma) replace the router.

## Consequences

- A second Go dependency: `oapi-codegen/runtime`, used by the generated parameter and body
  binding.
- New endpoints start in the YAML. That is about 15 lines per endpoint before any Go.
- Mismatches between Go and TypeScript become compile errors instead of runtime surprises.
- Unknown JSON fields are accepted (the generated decoder does not reject them).
