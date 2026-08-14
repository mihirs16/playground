package server

import (
	"encoding/json"
	"net/http"

	"github.com/mihirs16/playground/custodian/internal/api"
)

const problemContentType = "application/problem+json"

// writeProblem renders an RFC 9457 problem document with a stable machine
// code, the shape every custodian error takes so broom can branch on `code`.
func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	problem := api.Problem{
		Title:  http.StatusText(status),
		Status: status,
		Code:   code,
		Detail: &detail,
	}
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

// writeNotImplemented answers a route whose surrounding infrastructure —
// routing, auth, CORS, storage — is live but whose behaviour is not yet wired.
func writeNotImplemented(w http.ResponseWriter) {
	writeProblem(w, http.StatusNotImplemented, "not_implemented", "this endpoint is not implemented yet")
}
