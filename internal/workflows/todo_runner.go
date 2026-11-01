package workflows

import (
	"context"

	"github.com/2SSK/go-next-boilerplate/internal/activities"
	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// TodoRunner performs todo writes by running their workflows. It satisfies
// handlers.TodoWriter.
type TodoRunner struct {
	r *Runner
}

// NewTodoRunner returns a TodoRunner that runs workflows with r.
func NewTodoRunner(r *Runner) *TodoRunner {
	return &TodoRunner{r: r}
}

// Create runs CreateTodo.
func (t *TodoRunner) Create(ctx context.Context, owner, title string) (models.Todo, error) {
	var out models.Todo
	err := t.r.Run(ctx, "todo-create:"+owner, CreateTodo, activities.CreateTodoInput{Owner: owner, Title: title}, &out)
	return out, err
}

// Update runs UpdateTodo.
func (t *TodoRunner) Update(ctx context.Context, owner string, id int64, p models.TodoPatch) (models.Todo, error) {
	var out models.Todo
	err := t.r.Run(ctx, "todo-update:"+owner, UpdateTodo, activities.UpdateTodoInput{Owner: owner, ID: id, Patch: p}, &out)
	return out, err
}

// Delete runs DeleteTodo.
func (t *TodoRunner) Delete(ctx context.Context, owner string, id int64) error {
	return t.r.Run(ctx, "todo-delete:"+owner, DeleteTodo, activities.DeleteTodoInput{Owner: owner, ID: id}, nil)
}
