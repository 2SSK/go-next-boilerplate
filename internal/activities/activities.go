// Package activities holds Temporal activities: the steps of a workflow that
// touch the outside world (the database here). They may run more than once
// (Temporal retries them), so each must be safe to repeat or fail cleanly.
// Errors that retrying can't fix are returned as non-retryable application
// errors, with a Type that callers map back to domain errors.
package activities

// ErrTypeNotFound is the application error type for a missing record; the
// workflow runner maps it back to models.ErrNotFound.
const ErrTypeNotFound = "NotFound"
