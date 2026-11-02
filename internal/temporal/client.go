// Package temporal connects to the Temporal workflow engine. The API uses a
// lazy client (it starts even if Temporal is down; writes then fail with 503);
// the worker dials eagerly, since it has nothing to do without Temporal.
package temporal

import (
	"fmt"
	"log/slog"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	tlog "go.temporal.io/sdk/log"

	"github.com/2SSK/go-next-boilerplate/internal/config"
)

// options uses the global OpenTelemetry providers (internal/telemetry), so
// they must be set up first. The tracing interceptor carries the trace from
// the API request into the workflow and its activities on the worker.
func options(cfg config.Temporal, log *slog.Logger) (client.Options, error) {
	tracing, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		return client.Options{}, fmt.Errorf("temporal tracing: %w", err)
	}
	return client.Options{
		HostPort:     cfg.Address,
		Namespace:    cfg.Namespace,
		Logger:       tlog.NewStructuredLogger(log.With("component", "temporal")),
		Interceptors: []interceptor.ClientInterceptor{tracing},
		MetricsHandler: opentelemetry.NewMetricsHandler(opentelemetry.MetricsHandlerOptions{
			UseMonotonicCounters: true,
			OnError:              func(err error) { log.Warn("temporal metrics", "error", err) },
		}),
	}, nil
}

// NewLazyClient returns a client that connects on first use.
func NewLazyClient(cfg config.Temporal, log *slog.Logger) (client.Client, error) {
	opts, err := options(cfg, log)
	if err != nil {
		return nil, err
	}
	c, err := client.NewLazyClient(opts)
	if err != nil {
		return nil, fmt.Errorf("temporal client: %w", err)
	}
	return c, nil
}

// Dial connects now and fails if Temporal is unreachable.
func Dial(cfg config.Temporal, log *slog.Logger) (client.Client, error) {
	opts, err := options(cfg, log)
	if err != nil {
		return nil, err
	}
	c, err := client.Dial(opts)
	if err != nil {
		return nil, fmt.Errorf("dial temporal at %s: %w", cfg.Address, err)
	}
	return c, nil
}
