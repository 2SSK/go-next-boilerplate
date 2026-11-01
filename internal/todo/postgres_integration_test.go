//go:build integration

package todo_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/2SSK/go-next-boilerplate/internal/database"
	"github.com/2SSK/go-next-boilerplate/internal/todo"
)

// Runs against DATABASE_URL (make db-up first): go test -tags integration ./...
func TestPostgresStore(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://app:app@localhost:5432/app?sslmode=disable"
	}
	ctx := context.Background()

	db, err := database.Open(ctx, url, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer db.Close()

	s := todo.NewPostgresStore(db)

	created, err := s.Create(ctx, "integration test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), created.ID) })

	done := true
	updated, err := s.Update(ctx, created.ID, todo.Patch{Completed: &done})
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
	if _, err := s.Update(ctx, created.ID, todo.Patch{Completed: &done}); !errors.Is(err, todo.ErrNotFound) {
		t.Errorf("Update after delete = %v, want ErrNotFound", err)
	}
}
