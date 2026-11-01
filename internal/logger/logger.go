// Package logger builds the application's slog logger.
//
// Production writes one JSON object per line (for log collectors); everything
// else writes readable text. Every line carries service, version, and env.
// Lines logged with a request context (log.InfoContext(ctx, ...)) also carry
// that request's ID, so one request can be followed across every package.
// Attributes that look like secrets are redacted, and error lines record where
// they were logged.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
)

// Options configure New.
type Options struct {
	// Env is "production" (JSON) or anything else (text).
	Env string
	// Level is the minimum level written; see ParseLevel.
	Level   slog.Level
	Service string
	Version string
}

// New returns the logger described by o, writing to w.
func New(o Options, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: o.Level, ReplaceAttr: redact}

	var h slog.Handler
	if o.Env == "production" {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}

	return slog.New(contextHandler{h}).With(
		slog.String("service", o.Service),
		slog.String("version", o.Version),
		slog.String("env", o.Env),
	)
}

// ParseLevel reads LOG_LEVEL ("debug", "info", "warn", "error", any case).
// Empty means the default for env: info in production, debug elsewhere.
func ParseLevel(s, env string) (slog.Level, error) {
	if s == "" {
		if env == "production" {
			return slog.LevelInfo, nil
		}
		return slog.LevelDebug, nil
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error, got %q", s)
	}
	return l, nil
}

type requestIDKey struct{}

// WithRequestID returns ctx carrying id; lines logged with it include the id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the ID stored by WithRequestID, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// contextHandler adds what a record can't carry by itself: the request ID from
// its context, and the source line for errors.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	// Only errors pay for the source lookup; info lines stay cheap.
	if r.Level >= slog.LevelError && r.PC != 0 {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		r.AddAttrs(slog.String("source", fmt.Sprintf("%s:%d", trimPath(f.File), f.Line)))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

// trimPath keeps the last two path elements: "logger/logger.go".
func trimPath(file string) string {
	if i := strings.LastIndexByte(file, '/'); i > 0 {
		if j := strings.LastIndexByte(file[:i], '/'); j >= 0 {
			return file[j+1:]
		}
	}
	return file
}

// secretWords mark an attribute key as sensitive when they appear anywhere in
// it, case-insensitively: "password", "db_password", "Authorization", ...
var secretWords = []string{"password", "passwd", "secret", "token", "authorization", "cookie", "api_key", "apikey"}

// redact replaces the value of any sensitive attribute, at any group depth.
func redact(_ []string, a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	for _, w := range secretWords {
		if strings.Contains(key, w) {
			return slog.String(a.Key, "[REDACTED]")
		}
	}
	return a
}
