package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/temporal"

	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// TodoStore is the storage the todo activities need; store.Todos implements it.
type TodoStore interface {
	Create(ctx context.Context, owner, title string) (models.Todo, error)
	Update(ctx context.Context, owner string, id int64, p models.TodoPatch) (models.Todo, error)
	Delete(ctx context.Context, owner string, id int64) error
}

// Todos are the todo activities. Register one *Todos with the worker.
type Todos struct {
	store TodoStore
}

// NewTodos returns the todo activities backed by store.
func NewTodos(store TodoStore) *Todos {
	return &Todos{store: store}
}

// CreateTodoInput is the CreateTodo activity's argument.
type CreateTodoInput struct {
	Owner string
	Title string
}

// UpdateTodoInput is the UpdateTodo activity's argument.
type UpdateTodoInput struct {
	Owner string
	ID    int64
	Patch models.TodoPatch
}

// DeleteTodoInput is the DeleteTodo activity's argument.
type DeleteTodoInput struct {
	Owner string
	ID    int64
}

// CreateTodo inserts a todo.
func (a *Todos) CreateTodo(ctx context.Context, in CreateTodoInput) (models.Todo, error) {
	return a.store.Create(ctx, in.Owner, in.Title)
}

// UpdateTodo applies a patch. Setting the same fields twice is harmless, so a
// retry after a lost response is safe.
func (a *Todos) UpdateTodo(ctx context.Context, in UpdateTodoInput) (models.Todo, error) {
	t, err := a.store.Update(ctx, in.Owner, in.ID, in.Patch)
	return t, notFoundIsFinal(err)
}

// DeleteTodo removes a todo. A retry after a successful delete reports
// NotFound, which the caller treats as the todo being gone.
func (a *Todos) DeleteTodo(ctx context.Context, in DeleteTodoInput) error {
	return notFoundIsFinal(a.store.Delete(ctx, in.Owner, in.ID))
}

// notFoundIsFinal stops Temporal from retrying a missing record.
func notFoundIsFinal(err error) error {
	if errors.Is(err, models.ErrNotFound) {
		return temporal.NewNonRetryableApplicationError("todo not found", ErrTypeNotFound, err)
	}
	return err
}
