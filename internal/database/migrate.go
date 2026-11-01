package database

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations are plain SQL files, applied once each in name order (0001, 0002,
// ...) and never edited after they ship. Rules, enforced on every run:
//   - an applied file must still exist and hash to the checksum recorded for it
//   - a new file must sort after every applied one
//
// Each file runs in its own transaction together with its bookkeeping row, so
// a failure leaves the schema exactly as it was before that file.

//go:embed migrations/*.sql
var embedded embed.FS

// Migrations returns the migration files built into the binary.
func Migrations() fs.FS {
	sub, err := fs.Sub(embedded, "migrations")
	if err != nil {
		panic(err) // the directory is embedded at compile time
	}
	return sub
}

// migrationLockID is an arbitrary key for pg_advisory_lock, so replicas or
// deploy jobs that start together migrate one at a time instead of racing.
const migrationLockID = 7_331_001

// Migration is one SQL file.
type Migration struct {
	Version  string // the file name, e.g. "0002_todos.sql"
	Checksum string // hex SHA-256 of the file contents
	SQL      string
}

// Applied is a migration recorded in schema_migrations.
type Applied struct {
	Version   string
	Checksum  string
	AppliedAt time.Time
}

// Plan is the comparison of the files with the database.
type Plan struct {
	Applied []Applied
	Pending []Migration
}

// ErrNotCurrent is returned by CheckCurrent when migrations are pending.
var ErrNotCurrent = errors.New("database schema is not current")

// Load reads every *.sql file in fsys, sorted by name, with its checksum.
func Load(fsys fs.FS) ([]Migration, error) {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)

	files := make([]Migration, 0, len(names))
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		sum := sha256.Sum256(b)
		files = append(files, Migration{Version: path.Base(name), Checksum: hex.EncodeToString(sum[:]), SQL: string(b)})
	}
	return files, nil
}

// plan compares files with what the database recorded and enforces the rules
// above. It is pure, so every rule is unit-tested without a database.
func plan(files []Migration, applied []Applied) (Plan, error) {
	byVersion := make(map[string]Migration, len(files))
	for _, f := range files {
		byVersion[f.Version] = f
	}

	var problems []string
	last := ""
	for _, a := range applied {
		f, ok := byVersion[a.Version]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s was applied but its file is gone", a.Version))
		case f.Checksum != a.Checksum:
			problems = append(problems, fmt.Sprintf("%s was modified after it was applied", a.Version))
		}
		last = max(last, a.Version)
	}

	done := make(map[string]bool, len(applied))
	for _, a := range applied {
		done[a.Version] = true
	}
	p := Plan{Applied: applied}
	for _, f := range files {
		if done[f.Version] {
			continue
		}
		if f.Version < last {
			problems = append(problems, fmt.Sprintf("%s is new but sorts before the applied %s", f.Version, last))
			continue
		}
		p.Pending = append(p.Pending, f)
	}

	if len(problems) > 0 {
		return Plan{}, fmt.Errorf("migration history does not match the files: %s", strings.Join(problems, "; "))
	}
	return p, nil
}

// querier is satisfied by *pgx.Conn, *pgxpool.Pool, and pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// readApplied lists recorded migrations; none when the table doesn't exist yet.
func readApplied(ctx context.Context, q querier) ([]Applied, error) {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return nil, fmt.Errorf("look for schema_migrations: %w", err)
	}
	if !exists {
		return nil, nil
	}
	rows, _ := q.Query(ctx, `SELECT version, checksum, applied_at FROM schema_migrations ORDER BY version`)
	applied, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Applied])
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	return applied, nil
}

// Status reports applied and pending migrations without changing anything.
func Status(ctx context.Context, q querier, fsys fs.FS) (Plan, error) {
	files, err := Load(fsys)
	if err != nil {
		return Plan{}, err
	}
	applied, err := readApplied(ctx, q)
	if err != nil {
		return Plan{}, err
	}
	return plan(files, applied)
}

// CheckCurrent returns nil when every migration is applied and untouched. The
// API calls it at startup instead of migrating: schema changes are a deploy
// step (`api migrate`), never a side effect of starting a server.
func CheckCurrent(ctx context.Context, q querier, fsys fs.FS) error {
	p, err := Status(ctx, q, fsys)
	if err != nil {
		return err
	}
	if len(p.Pending) > 0 {
		names := make([]string, len(p.Pending))
		for i, m := range p.Pending {
			names[i] = m.Version
		}
		return fmt.Errorf("%w: %d pending (%s)", ErrNotCurrent, len(names), strings.Join(names, ", "))
	}
	return nil
}

// Migrate applies every pending migration and returns the ones it applied. It
// holds an advisory lock for the whole run, so concurrent runs wait and then
// find nothing left to do.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) ([]Migration, error) {
	files, err := Load(fsys)
	if err != nil {
		return nil, err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Advisory locks belong to a session, so lock and unlock on the same conn.
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return nil, fmt.Errorf("take migration lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			checksum   text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	// Read under the lock: another runner may have just finished.
	applied, err := readApplied(ctx, conn)
	if err != nil {
		return nil, err
	}
	p, err := plan(files, applied)
	if err != nil {
		return nil, err
	}

	for i, m := range p.Pending {
		err := pgx.BeginFunc(ctx, conn.Conn(), func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.SQL); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)`, m.Version, m.Checksum)
			return err
		})
		if err != nil {
			return p.Pending[:i], fmt.Errorf("apply %s (rolled back; earlier files stay applied): %w", m.Version, err)
		}
	}
	return p.Pending, nil
}
