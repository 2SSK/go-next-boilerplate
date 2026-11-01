// Package database owns the Postgres pool and the embedded migrations.
package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrationLockID is an arbitrary key for pg_advisory_lock, so replicas that
// boot together apply migrations one at a time instead of racing.
const migrationLockID = 7_331_001

// Open connects to Postgres, verifies the connection, and applies pending
// migrations. The caller owns the returned pool and must Close it.
func Open(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// pgxpool.New is lazy; ping now so a bad DATABASE_URL fails at startup.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	applied, err := migrate(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Info("database ready", "migrations_applied", applied)
	return pool, nil
}

// migrate applies embedded .sql files in name order (0001, 0002, ...), each
// in its own transaction, recording it in schema_migrations. Re-running is a
// no-op. It returns how many files were applied.
func migrate(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Advisory locks are per session, so lock and unlock on the same conn.
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return 0, fmt.Errorf("take migration lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationLockID) }()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return 0, fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return 0, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)

	applied := 0
	for _, name := range names {
		ok, err := applyOne(ctx, conn.Conn(), name)
		if err != nil {
			return applied, err
		}
		if ok {
			applied++
		}
	}
	return applied, nil
}

// applyOne runs a single migration file unless schema_migrations already lists
// it. The SQL and its bookkeeping row commit together or not at all.
func applyOne(ctx context.Context, conn *pgx.Conn, name string) (bool, error) {
	var done bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
	).Scan(&done); err != nil {
		return false, fmt.Errorf("check %s: %w", name, err)
	}
	if done {
		return false, nil
	}

	sql, err := migrationsFS.ReadFile(name)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", name, err)
	}

	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
