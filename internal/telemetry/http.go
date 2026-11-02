package telemetry

import (
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// HTTPHandler wraps h with a server span per request and the standard
// http.server.* metrics. Requests for which skip returns true (health probes)
// get neither.
func HTTPHandler(h http.Handler, skip func(*http.Request) bool) http.Handler {
	return otelhttp.NewHandler(h, "http.server",
		otelhttp.WithFilter(func(r *http.Request) bool { return !skip(r) }),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return r.Method }),
	)
}

// Route records the route the mux matched ("/api/v1/todos/{id}") on the
// request's span and metrics. HTTPHandler can't see it: it runs before the
// mux, on a different copy of the request. Wrap every route's handler.
func Route(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if route := routeOf(r.Pattern); route != "" {
			attr := semconv.HTTPRoute(route)
			span := trace.SpanFromContext(r.Context())
			span.SetName(r.Method + " " + route)
			span.SetAttributes(attr)
			if l, ok := otelhttp.LabelerFromContext(r.Context()); ok {
				l.Add(attr)
			}
		}
		next(w, r)
	}
}

// routeOf strips the method and host from a ServeMux pattern:
// "GET /todos/{id}" → "/todos/{id}".
func routeOf(pattern string) string {
	if i := strings.IndexByte(pattern, '/'); i >= 0 {
		return pattern[i:]
	}
	return ""
}
