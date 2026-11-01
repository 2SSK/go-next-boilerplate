// Package middleware provides the HTTP middleware chain: request IDs, access
// logs, and panic recovery.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/httpx"
	"github.com/2SSK/go-next-boilerplate/internal/logger"
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

// RequestIDHeader is read from the client when present and echoed back.
const RequestIDHeader = "X-Request-ID"

// RequestID makes sure every request carries an ID, in its context and in the
// response headers. Every line logged with that context includes it (see
// logger.WithRequestID), so a client report can be matched to the server logs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" || len(id) > 128 {
			id = newID()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(logger.WithRequestID(r.Context(), id)))
	})
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// statusWriter records the status code and body size, which
// http.ResponseWriter hides.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
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
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// probes are hit every few seconds by orchestrators and load balancers;
// successful ones are logged at debug so they don't drown real traffic.
var probes = map[string]bool{"/livez": true, "/readyz": true}

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
			case probes[r.URL.Path]:
				level = slog.LevelDebug
			}
			log.LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("route", r.Pattern),
				slog.Int("status", status),
				slog.Int("bytes", sw.bytes),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
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
					log.ErrorContext(r.Context(), "panic recovered",
						"panic", v,
						"stack", string(debug.Stack()),
					)
					httpx.Error(w, http.StatusInternalServerError, api.ErrorErrorCodeInternal, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimit caps request bodies at n bytes; reading past it fails, which the
// handlers report as a 400. JSON APIs rarely need more than a megabyte.
func BodyLimit(n int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
