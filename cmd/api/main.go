// Command api runs the HTTP API, or applies database migrations.
//
//	api                  serve HTTP (the default; same as `api serve`)
//	api migrate          apply pending migrations, then exit
//	api migrate -status  list applied and pending migrations, then exit
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"text/tabwriter"

	"github.com/2SSK/go-next-boilerplate/internal/auth"
	"github.com/2SSK/go-next-boilerplate/internal/config"
	"github.com/2SSK/go-next-boilerplate/internal/database"
	"github.com/2SSK/go-next-boilerplate/internal/logger"
	"github.com/2SSK/go-next-boilerplate/internal/router"
	"github.com/2SSK/go-next-boilerplate/internal/temporal"
	"github.com/2SSK/go-next-boilerplate/internal/workflows"
)

// version is set at build time (-ldflags "-X main.version=..."); otherwise it
// comes from the VCS stamp Go records in the binary, or "dev".
var version string

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log := logger.New(logger.Options{
		Env:     cfg.Env,
		Level:   cfg.LogLevel,
		Service: router.AppName,
		Version: buildVersion(),
	}, os.Stdout)
	// Also routes the standard library's log package through this logger.
	slog.SetDefault(log)

	// ctx is cancelled on SIGINT/SIGTERM, which starts the graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "serve":
		return serve(ctx, cfg, log)
	case "migrate":
		return migrate(ctx, cfg, log, args)
	default:
		return fmt.Errorf("unknown command %q (want serve or migrate)", cmd)
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// The server never changes the schema; it refuses to run against one that
	// is behind or has been tampered with.
	if err := database.CheckCurrent(ctx, db, database.Migrations()); err != nil {
		return fmt.Errorf("%w; run `make db-migrate` (or `api migrate`) first", err)
	}

	// Connects on first use, so the API starts (and serves reads) while
	// Temporal is down; writes answer 503 until it's back.
	tc, err := temporal.NewLazyClient(cfg.Temporal, log)
	if err != nil {
		return err
	}
	defer tc.Close()

	srv := router.New(cfg.Port, router.Routes(router.Deps{
		Log:         log,
		DB:          db,
		DocsEnabled: cfg.DocsEnabled,
		// Contacts the provider on first use, so the API can start before it.
		Auth: auth.NewVerifier(auth.Config{
			Issuer:       cfg.OIDC.Issuer,
			DiscoveryURL: cfg.OIDC.DiscoveryURL,
			Audience:     cfg.OIDC.Audience,
		}),
		Workflows: workflows.NewRunner(tc, cfg.Temporal.TaskQueue, 0),
	}))

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr)
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

func migrate(ctx context.Context, cfg config.Config, log *slog.Logger, args []string) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	status := flags.Bool("status", false, "list applied and pending migrations without changing anything")
	if err := flags.Parse(args); err != nil {
		return err
	}

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if *status {
		p, err := database.Status(ctx, db, database.Migrations())
		if err != nil {
			return err
		}
		return printStatus(os.Stdout, p)
	}

	applied, err := database.Migrate(ctx, db, database.Migrations())
	for _, m := range applied {
		log.Info("applied migration", "migration", m.Version)
	}
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		log.Info("schema already current")
	}
	return nil
}

func printStatus(w io.Writer, p database.Plan) error {
	// tabwriter buffers every row and writes on Flush, so a write error
	// surfaces there; the per-row returns carry nothing to check.
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "VERSION\tSTATE")
	for _, a := range p.Applied {
		_, _ = fmt.Fprintf(tw, "%s\tapplied %s\n", a.Version, a.AppliedAt.Format("2006-01-02 15:04:05"))
	}
	for _, m := range p.Pending {
		_, _ = fmt.Fprintf(tw, "%s\tpending\n", m.Version)
	}
	return tw.Flush()
}

// buildVersion reports the release this binary was built from.
func buildVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	var rev, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if len(rev) >= 12 {
		return rev[:12] + dirty
	}
	return "dev"
}
