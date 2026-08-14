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

// writeValidationProblem renders a 422 problem carrying the per-field reasons a
// write was rejected, so broom can point the author at the offending fields.
func writeValidationProblem(w http.ResponseWriter, fields []api.FieldError) {
	detail := "the request body failed validation"
	problem := api.Problem{
		Title:  http.StatusText(http.StatusUnprocessableEntity),
		Status: http.StatusUnprocessableEntity,
		Code:   "validation_failed",
		Detail: &detail,
		Errors: &fields,
	}
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(problem)
}

// writeNotImplemented answers a route whose surrounding infrastructure —
// routing, auth, CORS, storage — is live but whose behaviour is not yet wired.
func writeNotImplemented(w http.ResponseWriter) {
	writeProblem(w, http.StatusNotImplemented, "not_implemented", "this endpoint is not implemented yet")
}
