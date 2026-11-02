package workflows

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"

	"go.temporal.io/sdk/testsuite"

	"github.com/2SSK/go-next-boilerplate/internal/activities"
	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// flakyStore fails the first failures calls, then behaves like a store with
// one todo (ID 1) owned by alice.
type flakyStore struct {
	failures int32
	calls    atomic.Int32
}

func (s *flakyStore) fail() error {
	if s.calls.Add(1) <= s.failures {
		return errors.New("connection reset")
	}
	return nil
}

func (s *flakyStore) Create(_ context.Context, _, title string) (models.Todo, error) {
	if err := s.fail(); err != nil {
		return models.Todo{}, err
	}
	return models.Todo{ID: 2, Title: title}, nil
}

func (s *flakyStore) Update(_ context.Context, owner string, id int64, p models.TodoPatch) (models.Todo, error) {
	if err := s.fail(); err != nil {
		return models.Todo{}, err
	}
	if owner != "alice" || id != 1 {
		return models.Todo{}, models.ErrNotFound
	}
	return models.Todo{ID: 1, Title: "t", Completed: *p.Completed}, nil
}

func (s *flakyStore) Delete(_ context.Context, owner string, id int64) error {
	if err := s.fail(); err != nil {
		return err
	}
	if owner != "alice" || id != 1 {
		return models.ErrNotFound
	}
	return nil
}

func newEnv(t *testing.T, s *flakyStore) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(activities.NewTodos(s, slog.New(slog.DiscardHandler)))
	return env
}

func TestCreateTodo(t *testing.T) {
	t.Run("retries a transient failure and returns the todo", func(t *testing.T) {
		s := &flakyStore{failures: 2}
		env := newEnv(t, s)
		env.ExecuteWorkflow(CreateTodo, activities.CreateTodoInput{Owner: "alice", Title: "write tests"})

		if err := env.GetWorkflowError(); err != nil {
			t.Fatalf("workflow failed: %v", err)
		}
		var got models.Todo
		if err := env.GetWorkflowResult(&got); err != nil {
			t.Fatal(err)
		}
		if got.Title != "write tests" || got.ID != 2 {
			t.Errorf("got %+v", got)
		}
		if n := s.calls.Load(); n != 3 {
			t.Errorf("store called %d times, want 3 (two failures, one success)", n)
		}
	})

	t.Run("gives up after the retry limit", func(t *testing.T) {
		s := &flakyStore{failures: 100}
		env := newEnv(t, s)
		env.ExecuteWorkflow(CreateTodo, activities.CreateTodoInput{Owner: "alice", Title: "x"})

		if env.GetWorkflowError() == nil {
			t.Fatal("want an error")
		}
		if n := s.calls.Load(); n != todoActivity.RetryPolicy.MaximumAttempts {
			t.Errorf("store called %d times, want %d", n, todoActivity.RetryPolicy.MaximumAttempts)
		}
	})
}

func TestUpdateTodo(t *testing.T) {
	done := true
	tests := []struct {
		name     string
		in       activities.UpdateTodoInput
		wantErr  error
		wantCall int32
	}{
		{"updates the owner's todo", activities.UpdateTodoInput{Owner: "alice", ID: 1, Patch: models.TodoPatch{Completed: &done}}, nil, 1},
		{"someone else's todo is not found, without retries", activities.UpdateTodoInput{Owner: "bob", ID: 1, Patch: models.TodoPatch{Completed: &done}}, models.ErrNotFound, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &flakyStore{}
			env := newEnv(t, s)
			env.ExecuteWorkflow(UpdateTodo, tt.in)

			if err := domainError("test", env.GetWorkflowError()); !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if n := s.calls.Load(); n != tt.wantCall {
				t.Errorf("store called %d times, want %d", n, tt.wantCall)
			}
		})
	}
}

func TestDeleteTodo(t *testing.T) {
	t.Run("a missing todo maps to models.ErrNotFound", func(t *testing.T) {
		env := newEnv(t, &flakyStore{})
		env.ExecuteWorkflow(DeleteTodo, activities.DeleteTodoInput{Owner: "alice", ID: 99})

		if err := domainError("test", env.GetWorkflowError()); !errors.Is(err, models.ErrNotFound) {
			t.Fatalf("err = %v, want models.ErrNotFound", err)
		}
	})
}

func TestDomainError(t *testing.T) {
	t.Run("a timeout means unavailable", func(t *testing.T) {
		if err := domainError("id", context.DeadlineExceeded); !errors.Is(err, ErrUnavailable) {
			t.Errorf("err = %v, want ErrUnavailable", err)
		}
	})
	t.Run("anything else is passed on, wrapped", func(t *testing.T) {
		boom := errors.New("boom")
		err := domainError("id", boom)
		if !errors.Is(err, boom) || errors.Is(err, ErrUnavailable) {
			t.Errorf("err = %v", err)
		}
	})
}
