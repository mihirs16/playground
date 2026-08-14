package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// profileResponse is a profile record on the wire. Body is emitted verbatim as
// the raw JSON custodian stored, so the record round-trips without custodian
// ever imposing a shape on it.
type profileResponse struct {
	Key  string          `json:"key"`
	Body json.RawMessage `json:"body"`
}

// GetPublicProfile serves a profile record by key on the public surface,
// carrying the same ETag + revalidate-friendly Cache-Control treatment as logs
// and honouring If-None-Match → 304.
func (h *handlers) GetPublicProfile(w http.ResponseWriter, r *http.Request, key api.ProfileKey) {
	record, err := h.db.GetProfile(r.Context(), key)
	if errors.Is(err, storage.ErrProfileNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "no profile with that key")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the profile record")
		return
	}

	if serveConditional(w, r, profileETag(record)) {
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Key: record.Key, Body: json.RawMessage(record.Body)})
}

// PutProfile upserts a profile record. The body is opaque JSON custodian stores
// verbatim: it checks only that the payload is syntactically valid JSON, never
// its shape — that is a convention between broom and persona.
func (h *handlers) PutProfile(w http.ResponseWriter, r *http.Request, key api.ProfileKey) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the request body")
		return
	}
	if !json.Valid(body) {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}

	record, err := h.db.UpsertProfile(r.Context(), key, string(body))
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not write the profile record")
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Key: record.Key, Body: json.RawMessage(record.Body)})
}

// profileETag derives a read's validator from the key and stored body, so any
// edit to the body moves it.
func profileETag(record storage.Profile) string {
	return makeETag(record.Key, record.Body)
}
