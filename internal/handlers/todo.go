package handlers

import (
	"context"
	"errors"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/auth"
	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// TodoStore is the storage TodoHandler needs; store.Todos implements it and
// tests use an in-memory fake. Every call is scoped to one owner.
type TodoStore interface {
	List(ctx context.Context, owner string) ([]models.Todo, error)
	Create(ctx context.Context, owner, title string) (models.Todo, error)
	Update(ctx context.Context, owner string, id int64, p models.TodoPatch) (models.Todo, error)
	Delete(ctx context.Context, owner string, id int64) error
}

// errNoPrincipal means a todo operation ran without auth.Middleware, which the
// spec's security requirement guarantees; it surfaces as a 500, never as data.
var errNoPrincipal = errors.New("todo operation reached without an authenticated caller")

// owner is the caller's stable user ID.
func owner(ctx context.Context) (string, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok || p.Subject == "" {
		return "", errNoPrincipal
	}
	return p.Subject, nil
}

// TodoHandler implements the todo operations.
type TodoHandler struct {
	store TodoStore
}

// NewTodoHandler returns a TodoHandler backed by store.
func NewTodoHandler(store TodoStore) *TodoHandler {
	return &TodoHandler{store: store}
}

// ListTodos implements GET /api/v1/todos.
func (h *TodoHandler) ListTodos(ctx context.Context, _ api.ListTodosRequestObject) (api.ListTodosResponseObject, error) {
	who, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	todos, err := h.store.List(ctx, who)
	if err != nil {
		return nil, err
	}
	out := make(api.ListTodos200JSONResponse, len(todos))
	for i, t := range todos {
		out[i] = todoToAPI(t)
	}
	return out, nil
}

// CreateTodo implements POST /api/v1/todos.
func (h *TodoHandler) CreateTodo(ctx context.Context, req api.CreateTodoRequestObject) (api.CreateTodoResponseObject, error) {
	title, err := models.NormalizeTitle(req.Body.Title)
	if err != nil {
		return api.CreateTodo422JSONResponse{ValidationFailedJSONResponse: api.NewValidationFailed(err.Error())}, nil
	}
	who, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.store.Create(ctx, who, title)
	if err != nil {
		return nil, err
	}
	return api.CreateTodo201JSONResponse(todoToAPI(t)), nil
}

// UpdateTodo implements PATCH /api/v1/todos/{id}.
func (h *TodoHandler) UpdateTodo(ctx context.Context, req api.UpdateTodoRequestObject) (api.UpdateTodoResponseObject, error) {
	if req.Id < 1 {
		return api.UpdateTodo400JSONResponse{BadRequestJSONResponse: api.NewBadRequest("id must be a positive integer")}, nil
	}
	p := models.TodoPatch{Title: req.Body.Title, Completed: req.Body.Completed}
	if p.Title == nil && p.Completed == nil {
		return api.UpdateTodo422JSONResponse{ValidationFailedJSONResponse: api.NewValidationFailed("nothing to update: send title or completed")}, nil
	}
	if p.Title != nil {
		title, err := models.NormalizeTitle(*p.Title)
		if err != nil {
			return api.UpdateTodo422JSONResponse{ValidationFailedJSONResponse: api.NewValidationFailed(err.Error())}, nil
		}
		p.Title = &title
	}

	who, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.store.Update(ctx, who, req.Id, p)
	if errors.Is(err, models.ErrNotFound) {
		return api.UpdateTodo404JSONResponse{NotFoundJSONResponse: api.NewNotFound("todo not found")}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.UpdateTodo200JSONResponse(todoToAPI(t)), nil
}

// DeleteTodo implements DELETE /api/v1/todos/{id}.
func (h *TodoHandler) DeleteTodo(ctx context.Context, req api.DeleteTodoRequestObject) (api.DeleteTodoResponseObject, error) {
	if req.Id < 1 {
		return api.DeleteTodo400JSONResponse{BadRequestJSONResponse: api.NewBadRequest("id must be a positive integer")}, nil
	}
	who, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	err = h.store.Delete(ctx, who, req.Id)
	if errors.Is(err, models.ErrNotFound) {
		return api.DeleteTodo404JSONResponse{NotFoundJSONResponse: api.NewNotFound("todo not found")}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.DeleteTodo204Response{}, nil
}

func todoToAPI(t models.Todo) api.Todo {
	return api.Todo{Id: t.ID, Title: t.Title, Completed: t.Completed, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}
