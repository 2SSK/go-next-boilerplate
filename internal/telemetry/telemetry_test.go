package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRouteOf(t *testing.T) {
	tests := []struct{ pattern, want string }{
		{"GET /api/v1/todos/{id}", "/api/v1/todos/{id}"},
		{"/docs", "/docs"},
		{"GET example.com/x", "/x"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := routeOf(tt.pattern); got != tt.want {
				t.Errorf("routeOf(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestHTTPHandler(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	mux := http.NewServeMux()
	mux.HandleFunc("GET /todos/{id}", Route(func(http.ResponseWriter, *http.Request) {}))
	mux.HandleFunc("GET /livez", func(http.ResponseWriter, *http.Request) {})
	// A middleware that copies the request, as the real chain does: the route
	// must still reach the span.
	copying := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), copyKey{}, 1)))
	})
	h := HTTPHandler(copying, func(r *http.Request) bool { return r.URL.Path == "/livez" })

	for _, path := range []string{"/todos/7", "/livez"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1 (probes are skipped)", len(spans))
	}
	if got := spans[0].Name(); got != "GET /todos/{id}" {
		t.Errorf("span name = %q, want %q", got, "GET /todos/{id}")
	}
	var route string
	for _, a := range spans[0].Attributes() {
		if a.Key == "http.route" {
			route = a.Value.AsString()
		}
	}
	if route != "/todos/{id}" {
		t.Errorf("http.route = %q, want /todos/{id}", route)
	}
}

func TestSetupWithoutEndpoint(t *testing.T) {
	tel, err := Setup(context.Background(), Options{Service: "test", Version: "v0", Env: "development"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })

	t.Run("exports no logs", func(t *testing.T) {
		if tel.LogHandler() != nil {
			t.Error("LogHandler is set without an endpoint")
		}
	})

	t.Run("temporal latencies get second-scale buckets", func(t *testing.T) {
		h, err := otel.GetMeterProvider().Meter("temporal-sdk-go").Float64Histogram("temporal_test_latency", metric.WithUnit("s"))
		if err != nil {
			t.Fatal(err)
		}
		h.Record(context.Background(), 0.02)

		rec := httptest.NewRecorder()
		tel.MetricsServer(0).Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if !strings.Contains(rec.Body.String(), `le="0.025"`) {
			t.Errorf("temporal histogram lacks a 0.025s bucket:\n%s", grep(rec.Body.String(), "temporal_test_latency"))
		}
	})

	t.Run("serves metrics and liveness", func(t *testing.T) {
		srv := httptest.NewServer(tel.MetricsServer(0).Handler)
		defer srv.Close()

		for path, want := range map[string]string{"/metrics": "go_goroutine_count", "/livez": ""} {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
				t.Errorf("GET %s = %d, body missing %q", path, resp.StatusCode, want)
			}
		}
	})
}

type copyKey struct{}

func grep(s, sub string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
