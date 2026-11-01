# 6. Writes run as Temporal workflows

Status: accepted (2026-11-01), platform branch

## Context

The platform variant should show how a service hands work to Temporal, with the
retries, timeouts, and history that buys, using the todo demo rather than a
separate toy endpoint. Reads gain nothing from Temporal and must stay fast.

## Decision

- **Writes go through Temporal; reads don't.** `POST`, `PATCH`, and `DELETE
/api/v1/todos` each start a workflow (`internal/workflows`) whose single
  activity (`internal/activities`) calls the same store as before. `GET` reads
  Postgres directly. The handler takes a `TodoReader` and a `TodoWriter`, so it
  doesn't know which is which.
- **The request waits for the result** (up to 10s), so the API contract is
  unchanged: 201 with the todo, 404 for a missing one. The web app needs no
  polling.
- **A separate worker** (`cmd/worker`, same image as the API, entrypoint
  `worker`) runs workflows and activities. Scale it independently of the API.
- **Retries:** each attempt has 10s; up to 5 attempts, 200ms backoff doubling to
  5s. A missing record is a non-retryable `NotFound` error, which the API maps
  back to `models.ErrNotFound` (404).
- **Idempotency:** the workflow ID is `<op>:<user>:<X-Request-ID>`, with
  `USE_EXISTING` on conflict and `REJECT_DUPLICATE` on reuse. A client that
  retries with the same request ID joins the running workflow, or reads the
  result of the finished one, instead of writing twice. The user is part of the
  ID, so one user can't read another's result.
- **Failure:** if Temporal can't be reached, or no worker finishes in time, the
  write answers 503 `unavailable` (declared in the spec). The API's client is
  lazy, so the API still starts and serves reads while Temporal is down.
- **Development** uses the Temporal CLI's dev server in Compose (`make tp-up`,
  UI on :8233); it keeps no data across restarts.

## Consequences

- Every write is visible, with inputs, attempts, and errors, in the Temporal UI.
- A write takes a few extra milliseconds (start workflow, schedule activity).
- A 503 after a timeout doesn't mean the write failed: the workflow may still
  finish. Retrying with the same `X-Request-ID` returns its result.
- Production needs a real Temporal cluster (or Temporal Cloud) with persistence;
  set `TEMPORAL_ADDRESS` and `TEMPORAL_NAMESPACE`. TLS/API-key options go in
  `internal/temporal` when that happens.
- Workflow code must stay deterministic: no clock, randomness, or I/O outside
  activities. Changing a running workflow's logic needs versioning
  (`workflow.GetVersion`); these are short-lived, so redeploying after they
  drain is usually enough.
