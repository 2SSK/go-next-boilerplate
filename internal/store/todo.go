package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// Todos is the todos table.
type Todos struct {
	db *pgxpool.Pool
}

// NewTodos returns a Todos that reads and writes db.
func NewTodos(db *pgxpool.Pool) *Todos {
	return &Todos{db: db}
}

const todoColumns = `id, title, completed, created_at, updated_at`

// Every method takes the owner and filters by it: a todo that belongs to
// someone else behaves exactly like one that doesn't exist.

// List returns owner's todos, newest first.
func (s *Todos) List(ctx context.Context, owner string) ([]models.Todo, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+todoColumns+` FROM todos WHERE owner_id = $1 ORDER BY created_at DESC, id DESC`, owner)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	todos, err := pgx.CollectRows(rows, pgx.RowToStructByPos[models.Todo])
	if err != nil {
		return nil, fmt.Errorf("scan todos: %w", err)
	}
	return todos, nil
}

// Create inserts a todo for owner and returns it with its generated fields.
func (s *Todos) Create(ctx context.Context, owner, title string) (models.Todo, error) {
	rows, _ := s.db.Query(ctx,
		`INSERT INTO todos (owner_id, title) VALUES ($1, $2) RETURNING `+todoColumns, owner, title)
	t, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[models.Todo])
	if err != nil {
		return models.Todo{}, fmt.Errorf("create todo: %w", err)
	}
	return t, nil
}

// Update applies the non-nil fields of p. COALESCE keeps the current value
// for fields the patch leaves out.
func (s *Todos) Update(ctx context.Context, owner string, id int64, p models.TodoPatch) (models.Todo, error) {
	rows, _ := s.db.Query(ctx, `
		UPDATE todos
		SET title = COALESCE($2, title), completed = COALESCE($3, completed)
		WHERE id = $1 AND owner_id = $4
		RETURNING `+todoColumns, id, p.Title, p.Completed, owner)
	t, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[models.Todo])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Todo{}, models.ErrNotFound
	}
	if err != nil {
		return models.Todo{}, fmt.Errorf("update todo %d: %w", id, err)
	}
	return t, nil
}

// Delete removes a todo.
func (s *Todos) Delete(ctx context.Context, owner string, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM todos WHERE id = $1 AND owner_id = $2`, id, owner)
	if err != nil {
		return fmt.Errorf("delete todo %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}
