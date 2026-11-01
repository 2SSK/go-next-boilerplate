// Command worker runs Temporal workflows and activities: it polls the task
// queue the API starts workflows on, and does the work. Run as many copies as
// you need; Temporal spreads the work across them.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/worker"

	"github.com/2SSK/go-next-boilerplate/internal/activities" // feature:todo
	"github.com/2SSK/go-next-boilerplate/internal/config"
	"github.com/2SSK/go-next-boilerplate/internal/database"
	"github.com/2SSK/go-next-boilerplate/internal/logger"
	"github.com/2SSK/go-next-boilerplate/internal/store" // feature:todo
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
	log := logger.New(logger.Options{
		Env:     cfg.Env,
		Level:   cfg.LogLevel,
		Service: "worker",
		Version: version,
	}, os.Stdout)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	w.RegisterActivity(activities.NewTodos(store.NewTodos(db)))
	registered++
	// feature:todo:end

	if registered == 0 {
		// Temporal refuses to start a worker with nothing registered.
		log.Info("no workflows registered, idling", "task_queue", cfg.Temporal.TaskQueue)
		<-ctx.Done()
		return nil
	}

	log.Info("worker started", "task_queue", cfg.Temporal.TaskQueue, "namespace", cfg.Temporal.Namespace)
	// Stops polling on SIGINT/SIGTERM and lets running activities finish.
	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("worker: %w", err)
	}
	log.Info("worker stopped")
	return nil
}
