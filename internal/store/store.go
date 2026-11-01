// Package store reads and writes Postgres. Each type implements the interface
// its handler declares (handlers.TodoStore, ...), in terms of internal/models,
// and reports a missing record as models.ErrNotFound.
package store
