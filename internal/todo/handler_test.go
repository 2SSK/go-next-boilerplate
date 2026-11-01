package todo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// memStore is an in-memory Store for handler tests.
type memStore struct {
	mu     sync.Mutex
	todos  []Todo
	nextID int64
	err    error // when set, every call fails with it
}

func (m *memStore) List(context.Context) ([]Todo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Todo(nil), m.todos...), m.err
}

func (m *memStore) Create(_ context.Context, title string) (Todo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return Todo{}, m.err
	}
	m.nextID++
	t := Todo{ID: m.nextID, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.todos = append(m.todos, t)
	return t, nil
}

func (m *memStore) Update(_ context.Context, id int64, p Patch) (Todo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.todos {
		if m.todos[i].ID != id {
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
	return Todo{}, ErrNotFound
}

func (m *memStore) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.todos {
		if m.todos[i].ID == id {
			m.todos = append(m.todos[:i], m.todos[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func newTestMux(s Store) *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(s).Register(mux, "/todos")
	return mux
}

func do(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantTitle  string
	}{
		{"valid title is trimmed", `{"title":"  buy milk  "}`, http.StatusCreated, "buy milk"},
		{"empty title", `{"title":"   "}`, http.StatusUnprocessableEntity, ""},
		{"title too long", `{"title":"` + strings.Repeat("a", MaxTitleLen+1) + `"}`, http.StatusUnprocessableEntity, ""},
		{"malformed JSON", `{"title":`, http.StatusBadRequest, ""},
		{"unknown field", `{"title":"x","done":true}`, http.StatusBadRequest, ""},
		{"trailing data", `{"title":"x"}{}`, http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newTestMux(&memStore{}), http.MethodPost, "/todos", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantTitle == "" {
				return
			}
			var got Todo
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.Title != tt.wantTitle || got.ID == 0 {
				t.Errorf("todo = %+v, want title %q", got, tt.wantTitle)
			}
		})
	}
}

func TestList(t *testing.T) {
	t.Run("empty list encodes as []", func(t *testing.T) {
		rec := do(t, newTestMux(&memStore{}), http.MethodGet, "/todos", "")
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
			t.Errorf("got %d %q, want 200 []", rec.Code, rec.Body)
		}
	})

	t.Run("store failure is a generic 500", func(t *testing.T) {
		rec := do(t, newTestMux(&memStore{err: errors.New("pq: secret detail")}), http.MethodGet, "/todos", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("500 body leaks the store error: %s", rec.Body)
		}
	})
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{"toggle completed", "/todos/1", `{"completed":true}`, http.StatusOK},
		{"rename", "/todos/1", `{"title":"renamed"}`, http.StatusOK},
		{"empty patch", "/todos/1", `{}`, http.StatusUnprocessableEntity},
		{"blank title", "/todos/1", `{"title":" "}`, http.StatusUnprocessableEntity},
		{"missing todo", "/todos/99", `{"completed":true}`, http.StatusNotFound},
		{"non-numeric id", "/todos/abc", `{"completed":true}`, http.StatusBadRequest},
		{"zero id", "/todos/0", `{"completed":true}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &memStore{}
			_, _ = store.Create(context.Background(), "first")

			rec := do(t, newTestMux(store), http.MethodPatch, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	store := &memStore{}
	_, _ = store.Create(context.Background(), "first")
	mux := newTestMux(store)

	if rec := do(t, mux, http.MethodDelete, "/todos/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("first delete = %d, want 204", rec.Code)
	}
	if rec := do(t, mux, http.MethodDelete, "/todos/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", rec.Code)
	}
}
