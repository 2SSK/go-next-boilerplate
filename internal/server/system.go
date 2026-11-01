package server

import (
	"context"
	"net/http"
	"time"

	"github.com/2SSK/boilerplate/internal/httpx"
)

// Pinger reports whether a dependency is reachable. *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// RootResponse is served at GET / so a bare curl explains the API.
type RootResponse struct {
	Name   string `json:"name"`
	Health string `json:"health"`
}

// HealthResponse is served at GET /healthz.
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func handleRoot(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, RootResponse{Name: name, Health: "/healthz"})
	}
}

// handleHealth pings the database with a short deadline: 200 when it answers,
// 503 when it does not, so a load balancer stops routing to a broken replica.
func handleHealth(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "unavailable", Database: "down"})
			return
		}
		httpx.JSON(w, http.StatusOK, HealthResponse{Status: "ok", Database: "up"})
	}
}
