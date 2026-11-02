// Command worker runs Temporal workflows and activities: it polls the task
// queue the API starts workflows on, and does the work. Run as many copies as
// you need; Temporal spreads the work across them.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/worker"

	"github.com/2SSK/go-next-boilerplate/internal/activities" // feature:todo
	"github.com/2SSK/go-next-boilerplate/internal/config"
	"github.com/2SSK/go-next-boilerplate/internal/database"
	"github.com/2SSK/go-next-boilerplate/internal/logger"
	"github.com/2SSK/go-next-boilerplate/internal/store" // feature:todo
	"github.com/2SSK/go-next-boilerplate/internal/telemetry"
	"github.com/2SSK/go-next-boilerplate/internal/temporal"
	"github.com/2SSK/go-next-boilerplate/internal/workflows" // feature:todo
)

// version is set at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tel, log, err := setupTelemetry(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(sctx); err != nil {
			log.Warn("telemetry shutdown", "error", err)
		}
	}()
	defer serveMetrics(tel.MetricsServer(cfg.Telemetry.WorkerMetricsPort), log, stop)()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	if err := database.CheckCurrent(ctx, db, database.Migrations()); err != nil {
		return fmt.Errorf("%w; run `make db-migrate` (or `api migrate`) first", err)
	}

	tc, err := temporal.Dial(cfg.Temporal, log)
	if err != nil {
		return err
	}
	defer tc.Close()

	w := worker.New(tc, cfg.Temporal.TaskQueue, worker.Options{})
	registered := 0

	// feature:todo:begin
	w.RegisterWorkflow(workflows.CreateTodo)
	w.RegisterWorkflow(workflows.UpdateTodo)
	w.RegisterWorkflow(workflows.DeleteTodo)
	w.RegisterActivity(activities.NewTodos(store.NewTodos(db), log))
	registered++
	// feature:todo:end

	if registered == 0 {
		// Temporal refuses to start a worker with nothing registered.
		log.Info("no workflows registered, idling", "task_queue", cfg.Temporal.TaskQueue)
		<-ctx.Done()
		return nil
	}
	return runWorker(ctx, w, cfg.Temporal, log)
}

// setupTelemetry installs tracing and metrics, then the logger, which also
// exports through them.
func setupTelemetry(ctx context.Context, cfg config.Config) (*telemetry.Telemetry, *slog.Logger, error) {
	tel, err := telemetry.Setup(ctx, telemetry.Options{
		Namespace:    "go-next-boilerplate",
		Service:      "worker",
		Version:      version,
		Env:          cfg.Env,
		OTLPEndpoint: cfg.Telemetry.OTLPEndpoint,
	})
	if err != nil {
		return nil, nil, err
	}
	log := logger.New(logger.Options{
		Env:     cfg.Env,
		Level:   cfg.LogLevel,
		Service: "worker",
		Version: version,
		Export:  tel.LogHandler(),
	}, os.Stdout)
	slog.SetDefault(log)
	return tel, log, nil
}

// serveMetrics serves /metrics (Prometheus) and /livez (the healthcheck) in
// the background; if it can't, the worker stops. The returned func closes it.
func serveMetrics(srv *http.Server, log *slog.Logger, stop func()) func() {
	go func() {
		log.Info("metrics listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server", "error", err)
			stop()
		}
	}()
	return func() { _ = srv.Close() }
}

// runWorker polls until ctx ends (SIGINT/SIGTERM), then lets running
// activities finish.
func runWorker(ctx context.Context, w worker.Worker, cfg config.Temporal, log *slog.Logger) error {
	log.Info("worker started", "task_queue", cfg.TaskQueue, "namespace", cfg.Namespace)
	done := make(chan any)
	go func() { <-ctx.Done(); close(done) }()
	if err := w.Run(done); err != nil {
		return fmt.Errorf("worker: %w", err)
	}
	log.Info("worker stopped")
	return nil
}
