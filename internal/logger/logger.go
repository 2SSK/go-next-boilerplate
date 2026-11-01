// Package logger builds the application's slog logger.
package logger

import (
	"io"
	"log/slog"
)

// New returns a JSON logger in production (machines parse it) and a text
// logger everywhere else (people read it).
func New(env string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if env == "production" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	opts.Level = slog.LevelDebug
	return slog.New(slog.NewTextHandler(w, opts))
}
