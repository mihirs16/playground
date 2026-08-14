package server

import (
	"context"
	"net/http"
)

// healthy self-assesses custodian: SQLite reachable (SELECT 1) and the object
// store bucket reachable (HeadBucket). Third-party APIs are deliberately
// excluded — Steam being down must never turn custodian red. The result is
// recorded to the telemetry sink as the health gauge and returned.
func (s *Server) healthy(ctx context.Context) bool {
	ok := true
	if err := s.db.PingContext(ctx); err != nil {
		ok = false
	}
	if err := s.edges.ObjectStore.HeadBucket(ctx); err != nil {
		ok = false
	}
	s.edges.Telemetry.RecordHealth(ctx, ok)
	return ok
}

// healthz is the debug-only endpoint for a manual curl. It is not a
// load-bearing public contract — the health gauge exported over OTLP is.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if s.healthy(r.Context()) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("degraded"))
}
