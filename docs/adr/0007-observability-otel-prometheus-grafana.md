# 7. Observability: OpenTelemetry, Prometheus, and Grafana

Status: accepted (2026-11-02), platform branch

## Context

A write now crosses four processes (browser → Next.js → API → Temporal → worker →
Postgres). Without traces you can't tell where a slow request spent its time;
without metrics you can't see error rates or saturation; and logs are only
useful if you can find the ones for a given request.

## Decision

- **OpenTelemetry for traces and logs, everywhere.** The Go processes use the
  OTel SDK (`internal/telemetry`); Next.js uses `@vercel/otel`
  (`web/instrumentation.ts`). W3C trace context flows Next.js → API over HTTP and
  API → worker through Temporal's tracing interceptor, so one browser action is
  one trace. Instrumented: HTTP server (otelhttp, with the matched route),
  Postgres queries (otelpgx), workflows and activities (Temporal contrib).
- **Export is configuration.** Traces and logs go over OTLP/HTTP to
  `OTEL_EXPORTER_OTLP_ENDPOINT` when it's set, and nowhere when it isn't. Spans
  are still created without it, so log lines keep their `trace_id`.
- **Logs:** slog stays the API. Every line carries `trace_id`/`span_id`;
  redaction runs before the record fans out to stdout and to OTel, so secrets
  reach neither.
- **Metrics are pulled, in Prometheus format,** from a separate internal port
  per process (`METRICS_PORT` 9464 for the API, `WORKER_METRICS_PORT` 9465 for
  the worker), never the public API port. One OTel MeterProvider with the
  Prometheus exporter carries HTTP, pgx pool and query, Temporal SDK, Go runtime,
  and process metrics. Temporal's latency histograms get second-scale buckets;
  the SDK default suits milliseconds.
- **The local backend** is a second Compose file, `docker-compose.monitoring.yml`
  (profile `monitoring`, `make mon-up`): OTel Collector → Tempo (traces) and Loki
  (logs, via OTLP), Prometheus, Grafana with provisioned datasources and an
  overview dashboard, and the standalone Temporal UI. Grafana links metrics →
  traces (exemplars), traces → logs, and logs → traces.

## Consequences

- Swapping backends (Grafana Cloud, Honeycomb, Datadog, ...) means pointing
  `OTEL_EXPORTER_OTLP_ENDPOINT` elsewhere; application code doesn't change.
- Every request is traced (no sampling). At real traffic, sample in the
  collector (tail sampling keeps errors and slow traces) rather than in code.
- The monitoring stack is for development: anonymous Grafana admin, local
  storage, no retention policy, bound to 127.0.0.1. Production runs these as
  managed services or with their own HA setup.
- The Temporal dev server's UI is off; the UI comes with `make mon-up`.
