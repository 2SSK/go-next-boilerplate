// Command api runs the HTTP API.
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

	"github.com/2SSK/boilerplate/internal/config"
	"github.com/2SSK/boilerplate/internal/database"
	"github.com/2SSK/boilerplate/internal/logger"
	"github.com/2SSK/boilerplate/internal/server"
)

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

	log := logger.New(cfg.Env, os.Stdout)
	slog.SetDefault(log)

	// ctx is cancelled on SIGINT/SIGTERM, which starts the graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	srv := server.New(cfg.Port, server.Routes(server.Deps{Log: log, DB: db}))

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Info("shutting down")
	}

	// Stop accepting connections and let in-flight requests finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("shutdown complete")
	return nil
}
