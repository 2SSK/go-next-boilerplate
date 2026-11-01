//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
	// Unique owners, so runs against a shared dev database don't see each other.
	alice, bob := "alice-"+t.Name()+time.Now().Format("150405.000000"), "bob-"+t.Name()+time.Now().Format("150405.000000")

	created, err := s.Create(ctx, alice, "integration test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), alice, created.ID) })

	done := true
	updated, err := s.Update(ctx, alice, created.ID, models.TodoPatch{Completed: &done})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Completed || updated.Title != "integration test" {
		t.Errorf("Update kept wrong fields: %+v", updated)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updated_at trigger did not fire: %v -> %v", created.UpdatedAt, updated.UpdatedAt)
	}

	list, err := s.List(ctx, alice)
	if err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Errorf("List(alice) = %v, %v; want exactly her todo", list, err)
	}

	// Another owner can neither see nor touch it.
	if list, _ := s.List(ctx, bob); len(list) != 0 {
		t.Errorf("List(bob) = %v, want nothing", list)
	}
	if _, err := s.Update(ctx, bob, created.ID, models.TodoPatch{Completed: &done}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("Update by another owner = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, bob, created.ID); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("Delete by another owner = %v, want ErrNotFound", err)
	}

	if err := s.Delete(ctx, alice, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Update(ctx, alice, created.ID, models.TodoPatch{Completed: &done}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("Update after delete = %v, want ErrNotFound", err)
	}
}
