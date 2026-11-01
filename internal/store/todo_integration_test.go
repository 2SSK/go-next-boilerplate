//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/2SSK/go-next-boilerplate/internal/config"
	"github.com/2SSK/go-next-boilerplate/internal/database"
	"github.com/2SSK/go-next-boilerplate/internal/models"
	"github.com/2SSK/go-next-boilerplate/internal/store"
)

// Runs against the database from the environment (make db-up first):
// go test -tags integration ./...
func TestTodos(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer db.Close()
	if _, err := database.Migrate(ctx, db, database.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := store.NewTodos(db)

	created, err := s.Create(ctx, "integration test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), created.ID) })

	done := true
	updated, err := s.Update(ctx, created.ID, models.TodoPatch{Completed: &done})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Completed || updated.Title != "integration test" {
		t.Errorf("Update kept wrong fields: %+v", updated)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updated_at trigger did not fire: %v -> %v", created.UpdatedAt, updated.UpdatedAt)
	}

	list, err := s.List(ctx)
	if err != nil || len(list) == 0 || list[0].ID != created.ID {
		t.Errorf("List = %v, %v; want newest todo first", list, err)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Update(ctx, created.ID, models.TodoPatch{Completed: &done}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("Update after delete = %v, want ErrNotFound", err)
	}
}
