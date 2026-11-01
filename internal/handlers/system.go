package handlers

import (
	"context"
	"time"

	"github.com/2SSK/go-next-boilerplate/api"
)

// Pinger reports whether a dependency is reachable. *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// SystemHandler implements the API's own operations: GET /, /livez, /readyz.
type SystemHandler struct {
	name     string
	db       Pinger
	docsPath string // "" when the docs are turned off
}

// NewSystemHandler returns a SystemHandler reporting name, checking db, and
// pointing at docsPath (pass "" when the docs are off).
func NewSystemHandler(name string, db Pinger, docsPath string) *SystemHandler {
	return &SystemHandler{name: name, db: db, docsPath: docsPath}
}

// GetRoot names the API and points at the docs, so a bare curl explains it.
func (s *SystemHandler) GetRoot(context.Context, api.GetRootRequestObject) (api.GetRootResponseObject, error) {
	resp := api.GetRoot200JSONResponse{Name: s.name, Readiness: "/readyz"}
	if s.docsPath != "" {
		resp.Docs = &s.docsPath
	}
	return resp, nil
}

// GetLivez says the process is running. It checks nothing else, so a database
// outage never makes an orchestrator restart healthy API processes.
func (s *SystemHandler) GetLivez(context.Context, api.GetLivezRequestObject) (api.GetLivezResponseObject, error) {
	return api.GetLivez200JSONResponse{Status: api.LivenessStatusOk}, nil
}

// GetReadyz pings the database with a short deadline: 200 when it answers, 503
// when it doesn't, so a load balancer stops routing to a replica that can't
// serve.
func (s *SystemHandler) GetReadyz(ctx context.Context, _ api.GetReadyzRequestObject) (api.GetReadyzResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		return api.GetReadyz503JSONResponse{Status: api.ReadinessStatusUnavailable, Database: api.ReadinessDatabaseDown}, nil
	}
	return api.GetReadyz200JSONResponse{Status: api.ReadinessStatusOk, Database: api.ReadinessDatabaseUp}, nil
}
