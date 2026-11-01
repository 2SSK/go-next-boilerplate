package todo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the Store backed by the todos table.
type PostgresStore struct {
	db *pgxpool.Pool
}

// NewPostgresStore returns a Store that reads and writes db.
func NewPostgresStore(db *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: db}
}

const columns = `id, title, completed, created_at, updated_at`

// List returns every todo, newest first.
func (s *PostgresStore) List(ctx context.Context) ([]Todo, error) {
	rows, err := s.db.Query(ctx, `SELECT `+columns+` FROM todos ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	todos, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Todo])
	if err != nil {
		return nil, fmt.Errorf("scan todos: %w", err)
	}
	return todos, nil
}

// Create inserts a todo and returns it with its generated fields.
func (s *PostgresStore) Create(ctx context.Context, title string) (Todo, error) {
	rows, _ := s.db.Query(ctx, `INSERT INTO todos (title) VALUES ($1) RETURNING `+columns, title)
	t, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Todo])
	if err != nil {
		return Todo{}, fmt.Errorf("create todo: %w", err)
	}
	return t, nil
}

// Update applies the non-nil fields of p. COALESCE keeps the current value
// for fields the patch leaves out.
func (s *PostgresStore) Update(ctx context.Context, id int64, p Patch) (Todo, error) {
	rows, _ := s.db.Query(ctx, `
		UPDATE todos
		SET title = COALESCE($2, title), completed = COALESCE($3, completed)
		WHERE id = $1
		RETURNING `+columns, id, p.Title, p.Completed)
	t, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Todo])
	if errors.Is(err, pgx.ErrNoRows) {
		return Todo{}, ErrNotFound
	}
	if err != nil {
		return Todo{}, fmt.Errorf("update todo %d: %w", id, err)
	}
	return t, nil
}

// Delete removes a todo.
func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete todo %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
