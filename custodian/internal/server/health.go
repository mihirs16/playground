package server

import "net/http"

// healthz is the debug-only endpoint for a manual curl. It is not a
// load-bearing public contract — the health gauge exported over OTLP, computed
// on the poll loop's timer, is. It runs the same assessment on demand.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if s.health.Assess(r.Context()) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("degraded"))
}
