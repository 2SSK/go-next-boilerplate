// Package todo is the example feature: a CRUD resource from table to handler.
// Copy this package as the starting point for a new resource, or run
// `make reset` to remove it.
package todo

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxTitleLen matches the CHECK constraint in the todos migration.
const MaxTitleLen = 200

// ErrNotFound is returned by a Store when the todo does not exist.
var ErrNotFound = errors.New("todo not found")

// Todo is a single item on the list.
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Patch is a partial update; nil fields are left unchanged.
type Patch struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}

// Store persists todos. Handlers depend on this, not on Postgres, so tests
// can swap in an in-memory fake.
type Store interface {
	List(ctx context.Context) ([]Todo, error)
	Create(ctx context.Context, title string) (Todo, error)
	Update(ctx context.Context, id int64, p Patch) (Todo, error)
	Delete(ctx context.Context, id int64) error
}

// normalizeTitle trims surrounding space and enforces the length rules.
func normalizeTitle(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return "", errors.New("title is required")
	case utf8.RuneCountInString(s) > MaxTitleLen:
		return "", errors.New("title must be at most 200 characters")
	}
	return s, nil
}
