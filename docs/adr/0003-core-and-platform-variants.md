# 3. Two variants: core on `main`, extended on `platform`

Status: accepted (2026-11-01)

## Context

Some projects need only Go, Next.js, and Postgres (opslens, shipyard). Others need Keycloak,
Temporal, and observability (tenantflow). One template carrying everything would burden the
small projects; a template per project would duplicate the shared foundation.

## Decision

- `main` is the minimal core: the contract, migrations, testing, Docker, CI. It never depends on
  anything from `platform`.
- `platform` is a long-lived branch that extends the core with Keycloak, Temporal (a separate
  `cmd/worker`), OpenTelemetry and Prometheus/Grafana, and later Redis if a project needs it.
- Changes flow one way: core changes land on `main` and are merged into `platform`.
- `platform` adds files rather than editing core files (new packages, an extra make include,
  Compose profiles), so merges rarely conflict.
- CI runs on both branches, the reset check included. Releases are tagged per variant:
  `core/v1.0.0`, `platform/v1.0.0`.
- Proven code is extracted from tenantflow (patterns and implementations, not its architecture).
- Wiring stays explicit in each `main.go`; no generic plugin or provider layer.

If merging `main` into `platform` starts conflicting regularly, `platform` moves to its own
repository with the core as an upstream remote. Nothing else changes.

## Consequences

- New projects pick a variant at creation: the template button for core, a clone of the
  `platform` branch (then `make rename`) for the extended starter.
- Projects created from the template are snapshots; they don't sync with it afterwards. Pull a
  later fix by adding the template as a remote and cherry-picking.
