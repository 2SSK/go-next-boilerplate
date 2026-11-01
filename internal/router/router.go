// Package router is the HTTP entry point: it mounts every route generated from
// api/openapi.yaml onto one ServeMux, adds the docs (internal/docs) and a JSON
// 404, wraps it all in the middleware chain, and builds the http.Server.
package router

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2SSK/go-next-boilerplate/api"
	"github.com/2SSK/go-next-boilerplate/internal/auth"
	"github.com/2SSK/go-next-boilerplate/internal/docs"
	"github.com/2SSK/go-next-boilerplate/internal/handlers"
	"github.com/2SSK/go-next-boilerplate/internal/httpx"
	"github.com/2SSK/go-next-boilerplate/internal/middleware"
	"github.com/2SSK/go-next-boilerplate/internal/store" // feature:todo
	"github.com/2SSK/go-next-boilerplate/internal/workflows"
)

// AppName is reported by GET /.
const AppName = "go-next-boilerplate"

// maxBodyBytes caps request bodies; JSON APIs rarely need more.
const maxBodyBytes = 1 << 20

// Deps are the shared dependencies handlers are built from.
type Deps struct {
	Log *slog.Logger
	DB  *pgxpool.Pool
	// DocsEnabled mounts /docs and /openapi.yaml.
	DocsEnabled bool
	// Auth verifies access tokens for operations the spec secures.
	Auth auth.TokenVerifier
	// Workflows runs writes as Temporal workflows.
	Workflows *workflows.Runner
}

// Routes returns the full handler. Every route is defined in api/openapi.yaml;
// this wires each area's handler to its storage and mounts the result.
func Routes(d Deps) http.Handler {
	h := handlers.Handlers{
		SystemHandler: handlers.NewSystemHandler(AppName, d.DB, docsPath(d.DocsEnabled)),
		TodoHandler:   handlers.NewTodoHandler(store.NewTodos(d.DB), workflows.NewTodoRunner(d.Workflows)), // feature:todo
	}

	// Every failure path answers with the contract's error shape.
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
		// The body isn't valid JSON for the operation (or is too large).
		RequestErrorHandlerFunc: badRequest,
		// A handler returned an error: unexpected, so log it and say nothing
		// specific. Database details never reach the client.
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, workflows.ErrUnavailable) {
				d.Log.WarnContext(r.Context(), "workflow unavailable", "route", r.Pattern, "error", err)
				httpx.Error(w, http.StatusServiceUnavailable, api.ErrorErrorCodeUnavailable, "temporarily unavailable, try again")
				return
			}
			d.Log.ErrorContext(r.Context(), "request failed", "route", r.Pattern, "error", err)
			httpx.Error(w, http.StatusInternalServerError, api.ErrorErrorCodeInternal, "internal server error")
		},
	})

	mux := http.NewServeMux()
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseRouter: mux,
		// Runs inside each generated route, after it has recorded the
		// operation's security requirement, so the spec decides what's public.
		Middlewares: []api.MiddlewareFunc{auth.Middleware(d.Auth, d.Log)},
		// A path or query parameter doesn't parse (for example /todos/abc).
		ErrorHandlerFunc: badRequest,
	})
	if d.DocsEnabled {
		docs.Register(mux)
	}
	// Anything unmatched gets a JSON 404 instead of the mux's plain text.
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusNotFound, api.ErrorErrorCodeNotFound, "not found")
	})

	return middleware.Chain(mux,
		middleware.SecurityHeaders,
		middleware.RequestID,
		middleware.Logger(d.Log),
		middleware.Recover(d.Log),
		middleware.BodyLimit(maxBodyBytes),
	)
}

func docsPath(enabled bool) string {
	if enabled {
		return docs.Path
	}
	return ""
}

func badRequest(w http.ResponseWriter, _ *http.Request, err error) {
	httpx.Error(w, http.StatusBadRequest, api.ErrorErrorCodeBadRequest, err.Error())
}

// New returns an http.Server with timeouts set, so a slow client can never
// hold a connection open forever.
func New(port int, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
