//go:build integration

package database_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2SSK/go-next-boilerplate/internal/config"
	"github.com/2SSK/go-next-boilerplate/internal/database"
)

// freshDB creates an empty database for one test and drops it afterwards.
// Runs against the database from the environment (make db-up first):
// go test -tags integration ./...
func freshDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	admin, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(admin.Close)

	b := make([]byte, 4)
	_, _ = rand.Read(b)
	name := "migrate_test_" + hex.EncodeToString(b)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)") })

	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	u.Path = "/" + name
	pool, err := database.Open(ctx, u.String())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

var twoFiles = fstest.MapFS{
	"0001_a.sql": {Data: []byte("CREATE TABLE a (id int);")},
	"0002_b.sql": {Data: []byte("CREATE TABLE b (id int);")},
}

func TestMigrateFreshThenNoop(t *testing.T) {
	pool := freshDB(t)
	ctx := context.Background()

	if err := database.CheckCurrent(ctx, pool, twoFiles); !errors.Is(err, database.ErrNotCurrent) {
		t.Fatalf("CheckCurrent before migrating = %v, want ErrNotCurrent", err)
	}

	applied, err := database.Migrate(ctx, pool, twoFiles)
	if err != nil || len(applied) != 2 {
		t.Fatalf("first Migrate = %d applied, %v; want 2, nil", len(applied), err)
	}
	applied, err = database.Migrate(ctx, pool, twoFiles)
	if err != nil || len(applied) != 0 {
		t.Fatalf("second Migrate = %d applied, %v; want 0, nil", len(applied), err)
	}
	if err := database.CheckCurrent(ctx, pool, twoFiles); err != nil {
		t.Errorf("CheckCurrent after migrating = %v", err)
	}
}

func TestMigrateConcurrentRunnersApplyOnce(t *testing.T) {
	pool := freshDB(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	counts := make([]int, 5)
	errs := make([]error, 5)
	for i := range counts {
		wg.Go(func() {
			applied, err := database.Migrate(ctx, pool, twoFiles)
			counts[i], errs[i] = len(applied), err
		})
	}
	wg.Wait()

	total := 0
	for i := range counts {
		if errs[i] != nil {
			t.Fatalf("runner %d: %v", i, errs[i])
		}
		total += counts[i]
	}
	if total != 2 {
		t.Errorf("runners applied %d migrations in total, want exactly 2", total)
	}
}

func TestMigrateRejectsEditedHistory(t *testing.T) {
	pool := freshDB(t)
	ctx := context.Background()
	if _, err := database.Migrate(ctx, pool, twoFiles); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	edited := fstest.MapFS{
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id bigint);")},
		"0002_b.sql": twoFiles["0002_b.sql"],
	}
	if _, err := database.Migrate(ctx, pool, edited); err == nil || !strings.Contains(err.Error(), "modified after it was applied") {
		t.Errorf("Migrate with an edited file = %v, want a modified-history error", err)
	}
	if err := database.CheckCurrent(ctx, pool, edited); err == nil {
		t.Error("CheckCurrent accepted an edited file")
	}
}

func TestMigrateFailureRollsBackThatFileOnly(t *testing.T) {
	pool := freshDB(t)
	ctx := context.Background()
	broken := fstest.MapFS{
		"0001_ok.sql":     {Data: []byte("CREATE TABLE ok (id int);")},
		"0002_broken.sql": {Data: []byte("CREATE TABLE half (id int); SELECT * FROM missing_table;")},
	}

	applied, err := database.Migrate(ctx, pool, broken)
	if err == nil || len(applied) != 1 {
		t.Fatalf("Migrate = %d applied, %v; want 1 applied and an error", len(applied), err)
	}
	var half bool
	_ = pool.QueryRow(ctx, `SELECT to_regclass('half') IS NOT NULL`).Scan(&half)
	if half {
		t.Error("the failed file's first statement was kept; the file should roll back as a whole")
	}
	p, err := database.Status(ctx, pool, broken)
	if err != nil || len(p.Applied) != 1 || len(p.Pending) != 1 {
		t.Errorf("Status = %d applied, %d pending, %v; want 1, 1, nil", len(p.Applied), len(p.Pending), err)
	}
}

func TestEmbeddedMigrationsApply(t *testing.T) {
	pool := freshDB(t)
	if _, err := database.Migrate(context.Background(), pool, database.Migrations()); err != nil {
		t.Fatalf("the shipped migrations fail on a fresh database: %v", err)
	}
}
