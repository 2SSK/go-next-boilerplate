package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/2SSK/go-next-boilerplate/internal/activities"
	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// todoActivity bounds every todo activity: 10s per attempt, up to 5 attempts
// with backoff, and no retries for a record that doesn't exist.
var todoActivity = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        200 * time.Millisecond,
		BackoffCoefficient:     2,
		MaximumInterval:        5 * time.Second,
		MaximumAttempts:        5,
		NonRetryableErrorTypes: []string{activities.ErrTypeNotFound},
	},
}

// a is only used to name activity methods; the worker registers the real one.
var a *activities.Todos

// CreateTodo creates a todo.
func CreateTodo(ctx workflow.Context, in activities.CreateTodoInput) (models.Todo, error) {
	var t models.Todo
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, todoActivity), a.CreateTodo, in).Get(ctx, &t)
	return t, err
}

// UpdateTodo renames or completes a todo.
func UpdateTodo(ctx workflow.Context, in activities.UpdateTodoInput) (models.Todo, error) {
	var t models.Todo
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, todoActivity), a.UpdateTodo, in).Get(ctx, &t)
	return t, err
}

// DeleteTodo deletes a todo.
func DeleteTodo(ctx workflow.Context, in activities.DeleteTodoInput) error {
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, todoActivity), a.DeleteTodo, in).Get(ctx, nil)
}
