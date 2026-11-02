import { registerOTel } from "@vercel/otel";
import { API_URL } from "@/lib/api-url";

/**
 * Traces the Next.js server: every page render, route handler, and fetch gets a
 * span. Calls to the Go API carry the trace (W3C traceparent), so one trace runs
 * from here through the API, Temporal, and the worker. Spans are exported over
 * OTLP when OTEL_EXPORTER_OTLP_ENDPOINT is set (`make mon-up` runs a collector);
 * otherwise nothing leaves the process.
 */
export function register() {
  registerOTel({
    serviceName: "web",
    attributes: { "service.namespace": "go-next-boilerplate" },
    instrumentationConfig: {
      fetch: { propagateContextUrls: [API_URL] },
    },
  });
}
