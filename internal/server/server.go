// Package server builds the HTTP routing table and the http.Server around it.
package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2SSK/go-next-boilerplate/internal/httpx"
	"github.com/2SSK/go-next-boilerplate/internal/middleware"
	"github.com/2SSK/go-next-boilerplate/internal/todo" // feature:todo
)

// AppName is reported by GET /.
const AppName = "go-next-boilerplate"

// Deps are the shared dependencies handlers are built from.
type Deps struct {
	Log *slog.Logger
	DB  *pgxpool.Pool
}

// Routes returns the full handler: every route on one flat ServeMux, wrapped
// in the middleware chain. Go 1.22+ patterns match on method and path, so
// "GET /items" and "POST /items" are separate entries.
func Routes(d Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", handleRoot(AppName))
	mux.HandleFunc("GET /healthz", handleHealth(d.DB))
	// Anything unmatched gets a JSON 404 instead of the mux's plain text.
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusNotFound, "not found")
	})

	// feature:todo:begin
	todo.NewHandler(todo.NewPostgresStore(d.DB)).Register(mux, "/api/v1/todos")
	// feature:todo:end

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger(d.Log),
		middleware.Recover(d.Log),
	)
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
