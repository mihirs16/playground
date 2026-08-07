// Package server wires custodian's HTTP surface: one chi router that carries
// both the public /v1/* and admin /admin/v1/* sub-surfaces, guarded by
// prefix-scoped middleware, over a real SQLite database and the injected edges.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/config"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// Server is a constructed custodian: a router plus the dependencies its health
// check and handlers reach for.
type Server struct {
	http.Handler

	db    *storage.DB
	edges edges.Set
}

// New builds the whole HTTP handler. Middleware order is deliberate: recover
// wraps everything, then access logging, then the two prefix-scoped guards
// (CORS on /v1/*, auth on /admin/*). The generated router registers the two
// API surfaces onto the same base router.
func New(cfg config.Config, db *storage.DB, edgeSet edges.Set, logger *slog.Logger) *Server {
	srv := &Server{db: db, edges: edgeSet}

	router := chi.NewRouter()
	router.Use(recoverMiddleware(logger))
	router.Use(accessLogMiddleware(logger))
	router.Use(corsMiddleware(cfg.CORSAllowlist))
	router.Use(adminAuthMiddleware(cfg.AdminTokenHash))

	router.Get("/healthz", srv.healthz)

	api.HandlerFromMux(&handlers{db: db, edges: edgeSet}, router)

	srv.Handler = router
	return srv
}
