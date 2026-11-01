package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/auth"
	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// memStore is an in-memory TodoStore for handler tests, scoped by owner like
// the real one.
type memStore struct {
	mu     sync.Mutex
	todos  []models.Todo
	owners map[int64]string
	nextID int64
	err    error // when set, every call fails with it
}

func (m *memStore) List(_ context.Context, owner string) ([]models.Todo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []models.Todo
	for _, t := range m.todos {
		if m.owners[t.ID] == owner {
			out = append(out, t)
		}
	}
	return out, m.err
}

func (m *memStore) Create(_ context.Context, owner, title string) (models.Todo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return models.Todo{}, m.err
	}
	if m.owners == nil {
		m.owners = map[int64]string{}
	}
	m.nextID++
	t := models.Todo{ID: m.nextID, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.todos = append(m.todos, t)
	m.owners[t.ID] = owner
	return t, nil
}

func (m *memStore) Update(_ context.Context, owner string, id int64, p models.TodoPatch) (models.Todo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.todos {
		if m.todos[i].ID != id || m.owners[id] != owner {
			continue
		}
		if p.Title != nil {
			m.todos[i].Title = *p.Title
		}
		if p.Completed != nil {
			m.todos[i].Completed = *p.Completed
		}
		return m.todos[i], nil
	}
	return models.Todo{}, models.ErrNotFound
}

func (m *memStore) Delete(_ context.Context, owner string, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.todos {
		if m.todos[i].ID == id && m.owners[id] == owner {
			m.todos = append(m.todos[:i], m.todos[i+1:]...)
			return nil
		}
	}
	return models.ErrNotFound
}

// as returns a context signed in as subject, as auth.Middleware leaves it.
func as(subject string) context.Context {
	return auth.WithPrincipal(context.Background(), auth.Principal{Subject: subject, Roles: []string{"user"}})
}

func ptr[T any](v T) *T { return &v }

func TestCreateTodo(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantTitle string // set when the todo should be created
		wantErr   string // set when a 422 is expected
	}{
		{"valid title is trimmed", "  buy milk  ", "buy milk", ""},
		{"blank title", "   ", "", "title is required"},
		{"title too long", strings.Repeat("a", models.MaxTitleLen+1), "", "at most 200"},
		{"multibyte title at the limit", strings.Repeat("é", models.MaxTitleLen), strings.Repeat("é", models.MaxTitleLen), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewTodoHandler(&memStore{})
			resp, err := h.CreateTodo(as("alice"), api.CreateTodoRequestObject{Body: &api.NewTodo{Title: tt.title}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			checkCreate(t, resp, tt.wantTitle, tt.wantErr)
		})
	}
}

// checkCreate asserts a 201 with wantTitle, or a 422 mentioning wantErr.
func checkCreate(t *testing.T, resp api.CreateTodoResponseObject, wantTitle, wantErr string) {
	t.Helper()
	switch r := resp.(type) {
	case api.CreateTodo201JSONResponse:
		if wantTitle == "" || r.Title != wantTitle || r.Id == 0 {
			t.Errorf("created %+v, want title %q", r, wantTitle)
		}
	case api.CreateTodo422JSONResponse:
		if wantErr == "" || !strings.Contains(r.Error.Message, wantErr) || r.Error.Code != api.ErrorErrorCodeValidationFailed {
			t.Errorf("422 %+v, want message containing %q", r.Error, wantErr)
		}
	default:
		t.Fatalf("unexpected response %T", resp)
	}
}

func TestListTodos(t *testing.T) {
	t.Run("empty list is an empty array", func(t *testing.T) {
		resp, err := NewTodoHandler(&memStore{}).ListTodos(as("alice"), api.ListTodosRequestObject{})
		r, ok := resp.(api.ListTodos200JSONResponse)
		if err != nil || !ok || r == nil || len(r) != 0 {
			t.Errorf("got %#v, %v; want an empty, non-nil list", resp, err)
		}
	})

	t.Run("store failure is returned as an error, not a response", func(t *testing.T) {
		_, err := NewTodoHandler(&memStore{err: errors.New("pq: secret detail")}).ListTodos(as("alice"), api.ListTodosRequestObject{})
		if err == nil {
			t.Error("want the store error passed up, so the server logs it and answers a generic 500")
		}
	})
}

func TestUpdateTodo(t *testing.T) {
	tests := []struct {
		name string
		id   int64
		body api.TodoPatch
		want any // the expected response type
	}{
		{"toggle completed", 1, api.TodoPatch{Completed: ptr(true)}, api.UpdateTodo200JSONResponse{}},
		{"rename", 1, api.TodoPatch{Title: ptr(" renamed ")}, api.UpdateTodo200JSONResponse{}},
		{"empty patch", 1, api.TodoPatch{}, api.UpdateTodo422JSONResponse{}},
		{"blank title", 1, api.TodoPatch{Title: ptr(" ")}, api.UpdateTodo422JSONResponse{}},
		{"missing todo", 99, api.TodoPatch{Completed: ptr(true)}, api.UpdateTodo404JSONResponse{}},
		{"zero id", 0, api.TodoPatch{Completed: ptr(true)}, api.UpdateTodo400JSONResponse{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &memStore{}
			_, _ = store.Create(context.Background(), "alice", "first")

			resp, err := NewTodoHandler(store).UpdateTodo(as("alice"), api.UpdateTodoRequestObject{Id: tt.id, Body: &tt.body})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fmt.Sprintf("%T", resp) != fmt.Sprintf("%T", tt.want) {
				t.Fatalf("response %T, want %T", resp, tt.want)
			}
			if r, ok := resp.(api.UpdateTodo200JSONResponse); ok && tt.body.Title != nil && r.Title != "renamed" {
				t.Errorf("title %q, want it trimmed to %q", r.Title, "renamed")
			}
		})
	}
}

func TestDeleteTodo(t *testing.T) {
	store := &memStore{}
	_, _ = store.Create(context.Background(), "alice", "first")
	h := NewTodoHandler(store)
	ctx := as("alice")

	if resp, _ := h.DeleteTodo(ctx, api.DeleteTodoRequestObject{Id: 1}); fmt.Sprintf("%T", resp) != "api.DeleteTodo204Response" {
		t.Fatalf("first delete = %T, want 204", resp)
	}
	if resp, _ := h.DeleteTodo(ctx, api.DeleteTodoRequestObject{Id: 1}); fmt.Sprintf("%T", resp) != "api.DeleteTodo404JSONResponse" {
		t.Errorf("second delete = %T, want 404", resp)
	}
}

func TestTodosAreIsolatedPerUser(t *testing.T) {
	store := &memStore{}
	h := NewTodoHandler(store)
	created, err := h.CreateTodo(as("alice"), api.CreateTodoRequestObject{Body: &api.NewTodo{Title: "alice's"}})
	if err != nil {
		t.Fatal(err)
	}
	id := created.(api.CreateTodo201JSONResponse).Id

	if list, _ := h.ListTodos(as("bob"), api.ListTodosRequestObject{}); len(list.(api.ListTodos200JSONResponse)) != 0 {
		t.Errorf("bob sees alice's todos: %v", list)
	}
	upd, _ := h.UpdateTodo(as("bob"), api.UpdateTodoRequestObject{Id: id, Body: &api.TodoPatch{Completed: ptr(true)}})
	if _, ok := upd.(api.UpdateTodo404JSONResponse); !ok {
		t.Errorf("bob updating alice's todo got %T, want a 404 (not a 403: existence isn't revealed)", upd)
	}
	del, _ := h.DeleteTodo(as("bob"), api.DeleteTodoRequestObject{Id: id})
	if _, ok := del.(api.DeleteTodo404JSONResponse); !ok {
		t.Errorf("bob deleting alice's todo got %T, want a 404", del)
	}
	if list, _ := h.ListTodos(as("alice"), api.ListTodosRequestObject{}); len(list.(api.ListTodos200JSONResponse)) != 1 {
		t.Error("alice's todo did not survive bob's attempts")
	}
}

func TestTodoOperationsRefuseAnonymousCallers(t *testing.T) {
	// Only reachable if a route lost its security requirement; must fail closed.
	h := NewTodoHandler(&memStore{})
	if _, err := h.ListTodos(context.Background(), api.ListTodosRequestObject{}); !errors.Is(err, errNoPrincipal) {
		t.Errorf("ListTodos without a caller: err = %v, want errNoPrincipal", err)
	}
}
