// Package temporal connects to the Temporal workflow engine. The API uses a
// lazy client (it starts even if Temporal is down; writes then fail with 503);
// the worker dials eagerly, since it has nothing to do without Temporal.
package temporal

import (
	"fmt"
	"log/slog"

	"go.temporal.io/sdk/client"
	tlog "go.temporal.io/sdk/log"

	"github.com/2SSK/go-next-boilerplate/internal/config"
)

func options(cfg config.Temporal, log *slog.Logger) client.Options {
	return client.Options{
		HostPort:  cfg.Address,
		Namespace: cfg.Namespace,
		Logger:    tlog.NewStructuredLogger(log.With("component", "temporal")),
	}
}

// NewLazyClient returns a client that connects on first use.
func NewLazyClient(cfg config.Temporal, log *slog.Logger) (client.Client, error) {
	c, err := client.NewLazyClient(options(cfg, log))
	if err != nil {
		return nil, fmt.Errorf("temporal client: %w", err)
	}
	return c, nil
}

// Dial connects now and fails if Temporal is unreachable.
func Dial(cfg config.Temporal, log *slog.Logger) (client.Client, error) {
	c, err := client.Dial(options(cfg, log))
	if err != nil {
		return nil, fmt.Errorf("dial temporal at %s: %w", cfg.Address, err)
	}
	return c, nil
}
