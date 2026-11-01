# 2. Forward-only migrations, run as a deploy step

Status: accepted (2026-11-01)

## Context

The API used to apply migrations on startup. With several replicas, that makes every rollout a
schema change, and a failed migration a failed boot. An edited, already-applied file went
unnoticed. The author has been bitten by migration failures in production before.

## Decision

- Migrations are plain SQL files, applied in name order, each in its own transaction together
  with its bookkeeping row in `schema_migrations` (version, SHA-256 checksum, time).
- They run through a dedicated command, `api migrate` (`make db-migrate`; a one-shot `migrate`
  service in Compose; a Job or release step elsewhere), under a Postgres advisory lock.
- The server never changes the schema. At startup it checks that nothing is pending and nothing
  applied was edited, and refuses to start otherwise.
- Rules enforced on every run: an applied file must still exist with the same checksum; a new
  file must sort after every applied one. History is never rewritten automatically.
- Forward-only: no down migrations. To undo a change, write a new migration. Rolling back schema
  automatically risks data, and most real reversals need a data decision anyway.

## Consequences

- Deploys have an explicit migrate step before new API versions start.
- Statements that cannot run in a transaction (`CREATE INDEX CONCURRENTLY`) need their own
  approach when the time comes.
- A migration that fails leaves earlier files applied and its own changes rolled back; fix it
  with a new file, or edit it only if it never reached a shared database.
