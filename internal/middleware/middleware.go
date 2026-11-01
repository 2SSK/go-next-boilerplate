// Package middleware provides the HTTP middleware chain: request IDs, access
// logs, and panic recovery.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/2SSK/go-next-boilerplate/internal/httpx"
)

// Middleware wraps a handler with extra behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares so the first one listed runs first.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type ctxKey struct{}

// RequestIDHeader is read from the client when present and echoed back.
const RequestIDHeader = "X-Request-ID"

// RequestID makes sure every request carries an ID, in its context and in the
// response headers, so a log line can be matched to a client report.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" || len(id) > 128 {
			id = newID()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

// GetRequestID returns the request ID stored by RequestID, or "".
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// statusWriter records the status code, which http.ResponseWriter hides.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Logger writes one structured line per request. The route label is the
// matched mux pattern, so /items/1 and /items/2 group under one route.
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)

			status := sw.status
			if status == 0 {
				status = http.StatusOK
			}
			level := slog.LevelInfo
			switch {
			case status >= 500:
				level = slog.LevelError
			case status >= 400:
				level = slog.LevelWarn
			}
			log.LogAttrs(r.Context(), level, "http request",
				slog.String("request_id", GetRequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("route", r.Pattern),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

// Recover turns a handler panic into a 500 instead of a dropped connection.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					// ErrAbortHandler is net/http's deliberate "drop this connection".
					if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(v)
					}
					log.Error("panic recovered",
						"request_id", GetRequestID(r.Context()),
						"panic", v,
					)
					httpx.Error(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
