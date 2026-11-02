package activities

import (
	"context"
	"errors"
	"log/slog"

	"go.temporal.io/sdk/activity"
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
	log   *slog.Logger
}

// NewTodos returns the todo activities backed by store, logging to log.
func NewTodos(store TodoStore, log *slog.Logger) *Todos {
	return &Todos{store: store, log: log}
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
	t, err := a.store.Create(ctx, in.Owner, in.Title)
	a.done(ctx, "create todo", t.ID, err)
	return t, err
}

// UpdateTodo applies a patch. Setting the same fields twice is harmless, so a
// retry after a lost response is safe.
func (a *Todos) UpdateTodo(ctx context.Context, in UpdateTodoInput) (models.Todo, error) {
	t, err := a.store.Update(ctx, in.Owner, in.ID, in.Patch)
	a.done(ctx, "update todo", in.ID, err)
	return t, notFoundIsFinal(err)
}

// DeleteTodo removes a todo. A retry after a successful delete reports
// NotFound, which the caller treats as the todo being gone.
func (a *Todos) DeleteTodo(ctx context.Context, in DeleteTodoInput) error {
	err := a.store.Delete(ctx, in.Owner, in.ID)
	a.done(ctx, "delete todo", in.ID, err)
	return notFoundIsFinal(err)
}

// done logs an attempt's outcome. ctx carries the activity's span, so the
// line has the same trace_id as the API request that started the workflow.
func (a *Todos) done(ctx context.Context, op string, id int64, err error) {
	info := activity.GetInfo(ctx)
	attrs := []any{"todo_id", id, "workflow_id", info.WorkflowExecution.ID, "attempt", info.Attempt}
	switch {
	case err == nil:
		a.log.InfoContext(ctx, op, attrs...)
	case errors.Is(err, models.ErrNotFound):
		a.log.InfoContext(ctx, op+": not found", attrs...)
	default:
		// Temporal retries it; the last attempt's error reaches the API.
		a.log.WarnContext(ctx, op+" failed", append(attrs, "error", err)...)
	}
}

// notFoundIsFinal stops Temporal from retrying a missing record.
func notFoundIsFinal(err error) error {
	if errors.Is(err, models.ErrNotFound) {
		return temporal.NewNonRetryableApplicationError("todo not found", ErrTypeNotFound, err)
	}
	return err
}
