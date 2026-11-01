// Package store reads and writes Postgres. Each type implements the interfaces
// its callers declare (handlers.TodoReader for reads, activities.TodoStore for
// writes), in terms of internal/models, and reports a missing record as
// models.ErrNotFound.
package store
