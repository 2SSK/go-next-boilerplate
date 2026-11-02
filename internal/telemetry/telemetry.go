// Package telemetry sets up OpenTelemetry for a process: traces and logs go to
// an OTLP collector when OTEL_EXPORTER_OTLP_ENDPOINT is set, and metrics are
// served in Prometheus format on a separate, internal port (see MetricsServer).
//
// Without an endpoint nothing leaves the process, but spans are still created,
// so log lines carry a trace_id and trace context still reaches Temporal.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Options describe the process and where its telemetry goes.
type Options struct {
	// Namespace groups the app's services (api, worker, web) in backends.
	Namespace string
	Service   string
	Version   string
	Env       string
	// OTLPEndpoint is the collector's OTLP/HTTP base URL
	// (OTEL_EXPORTER_OTLP_ENDPOINT, e.g. http://localhost:4318). Empty keeps
	// traces and logs in-process.
	OTLPEndpoint string
}

// Telemetry owns the providers Setup installed. Call Shutdown on exit so
// buffered spans and logs are flushed.
type Telemetry struct {
	registry *prometheus.Registry
	logs     *sdklog.LoggerProvider // nil when not exporting
	shutdown []func(context.Context) error
}

// Setup installs the global tracer, meter, and propagator for o.
func Setup(ctx context.Context, o Options) (*Telemetry, error) {
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceNamespace(o.Namespace),
		semconv.ServiceName(o.Service),
		semconv.ServiceVersion(o.Version),
		attribute.String("deployment.environment.name", o.Env),
	))
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}
	t := &Telemetry{registry: prometheus.NewRegistry()}

	// W3C trace context in and out: Next.js → API → Temporal → worker.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetErrorHandler(exportErrors{})

	tracerOpts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if o.OTLPEndpoint != "" {
		// The exporters read OTEL_EXPORTER_OTLP_ENDPOINT (and the other
		// standard OTEL_* variables) themselves.
		exp, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("trace exporter: %w", err)
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(exp))

		logExp, err := otlploghttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("log exporter: %w", err)
		}
		t.logs = sdklog.NewLoggerProvider(sdklog.WithResource(res), sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)))
		t.shutdown = append(t.shutdown, t.logs.Shutdown)
	}
	tp := sdktrace.NewTracerProvider(tracerOpts...)
	otel.SetTracerProvider(tp)
	t.shutdown = append(t.shutdown, tp.Shutdown)

	// Metrics are pulled by Prometheus from MetricsServer, not pushed.
	t.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reader, err := otelprom.New(otelprom.WithRegisterer(t.registry))
	if err != nil {
		return nil, fmt.Errorf("prometheus exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(reader), sdkmetric.WithView(temporalLatencyView))
	otel.SetMeterProvider(mp)
	t.shutdown = append(t.shutdown, mp.Shutdown)
	// Go runtime metrics: goroutines, heap, GC.
	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, fmt.Errorf("runtime metrics: %w", err)
	}
	return t, nil
}

// secondBuckets are histogram boundaries for latencies measured in seconds,
// matching the ones the HTTP and database instrumentation use.
var secondBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10, 30, 60}

// temporalLatencyView gives the Temporal SDK's latency histograms (recorded in
// seconds, without bucket hints) second-scale buckets. The defaults
// (0, 5, 10, 25, ...) suit milliseconds and would put every request in the
// first bucket.
func temporalLatencyView(i sdkmetric.Instrument) (sdkmetric.Stream, bool) {
	if i.Scope.Name != "temporal-sdk-go" || i.Kind != sdkmetric.InstrumentKindHistogram {
		return sdkmetric.Stream{}, false
	}
	return sdkmetric.Stream{
		Name:        i.Name,
		Description: i.Description,
		Unit:        i.Unit,
		Aggregation: sdkmetric.AggregationExplicitBucketHistogram{Boundaries: secondBuckets},
	}, true
}

// LogHandler returns an slog.Handler that sends records to the collector, or
// nil when logs aren't exported.
func (t *Telemetry) LogHandler() slog.Handler {
	if t.logs == nil {
		return nil
	}
	return otelslog.NewHandler("app", otelslog.WithLoggerProvider(t.logs))
}

// Shutdown flushes and stops every provider.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error
	for _, f := range t.shutdown {
		errs = append(errs, f(ctx))
	}
	return errors.Join(errs...)
}

// MetricsServer serves /metrics (Prometheus) and /livez on port. Keep this port
// internal: it is for Prometheus and the orchestrator, never the public.
func (t *Telemetry) MetricsServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(t.registry, promhttp.HandlerOpts{
		// OpenMetrics carries exemplars: trace IDs on latency buckets.
		EnableOpenMetrics: true,
	}))
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// exportErrors logs OpenTelemetry's own errors (usually: the collector isn't
// running) at most once a minute, so a missing collector is visible without
// flooding the log.
type exportErrors struct{}

var (
	exportMu   sync.Mutex
	exportLast time.Time
)

func (exportErrors) Handle(err error) {
	exportMu.Lock()
	defer exportMu.Unlock()
	if time.Since(exportLast) < time.Minute {
		return
	}
	exportLast = time.Now()
	slog.Warn("telemetry export failing; is the collector running (make mon-up)?", "error", err)
}
