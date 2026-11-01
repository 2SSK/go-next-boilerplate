// Package workflows holds Temporal workflow definitions and the Runner the API
// starts them with. Workflow code must be deterministic: it only calls
// activities (internal/activities) and the workflow package's own APIs, never
// the clock, randomness, or I/O directly.
package workflows

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/2SSK/go-next-boilerplate/internal/activities"
	"github.com/2SSK/go-next-boilerplate/internal/logger"
	"github.com/2SSK/go-next-boilerplate/internal/models"
)

// ErrUnavailable means the write couldn't be confirmed: Temporal is
// unreachable, or no worker finished the workflow in time. The API answers 503.
var ErrUnavailable = errors.New("workflow engine unavailable")

// DefaultWait bounds how long a request waits for its workflow.
const DefaultWait = 10 * time.Second

// Runner starts workflows on one task queue and waits for their results.
type Runner struct {
	client    client.Client
	taskQueue string
	wait      time.Duration
}

// NewRunner returns a Runner that starts workflows on taskQueue and waits up
// to wait for each (DefaultWait if zero).
func NewRunner(c client.Client, taskQueue string, wait time.Duration) *Runner {
	if wait <= 0 {
		wait = DefaultWait
	}
	return &Runner{client: c, taskQueue: taskQueue, wait: wait}
}

// Run executes wf(in) and stores its result in out (nil for none).
//
// The workflow ID is key plus the request ID, so a retried request (same
// X-Request-ID) gets the original result instead of doing the work twice: a
// running workflow is joined, a finished one is read back. key should name the
// operation and its caller ("todo-create:<user>"), so nobody can read another
// user's result by reusing their request ID.
func (r *Runner) Run(ctx context.Context, key string, wf, in, out any) error {
	ctx, cancel := context.WithTimeout(ctx, r.wait)
	defer cancel()

	id := key + ":" + requestKey(ctx)
	run, err := r.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       id,
		TaskQueue:                r.taskQueue,
		WorkflowExecutionTimeout: time.Minute,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}, wf, in)
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	switch {
	case errors.As(err, &started):
		run = r.client.GetWorkflow(ctx, id, "") // already finished: read its result
	case err != nil:
		// Starting only fails when the server can't be reached or refuses us.
		return fmt.Errorf("%w: start %s: %w", ErrUnavailable, id, err)
	}
	return domainError(id, run.Get(ctx, out))
}

// domainError maps what a workflow run returned back to the domain.
func domainError(id string, err error) error {
	var app *temporal.ApplicationError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &app) && app.Type() == activities.ErrTypeNotFound:
		return models.ErrNotFound
	case errors.Is(err, context.DeadlineExceeded), isUnavailable(err):
		// The workflow may still finish; a retry with the same request ID
		// will return its result.
		return fmt.Errorf("%w: wait for %s: %w", ErrUnavailable, id, err)
	default:
		return fmt.Errorf("workflow %s: %w", id, err)
	}
}

func isUnavailable(err error) bool {
	var u *serviceerror.Unavailable
	return errors.As(err, &u)
}

// requestKey is the request's ID, or a random one when there isn't any.
func requestKey(ctx context.Context) string {
	if id := logger.RequestID(ctx); id != "" {
		return id
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
