package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// integrationCacheControl is the read surface's short TTL: persona's widgets get
// near-live status with a 60-second cache and a 5-minute stale-while-revalidate
// grace, paired with an ETag so a revalidation costs a 304, not a full body.
const integrationCacheControl = "public, max-age=60, s-maxage=60, stale-while-revalidate=300"

// GetPublicIntegration serves the latest stored row for a source as
// last-known-good: its data and the timestamp custodian last fetched it, never
// an error. A source not yet polled successfully returns a legal
// empty-but-present shape (null data, zero timestamp) rather than a 404.
func (h *handlers) GetPublicIntegration(w http.ResponseWriter, r *http.Request, source api.Source) {
	record, err := h.db.LatestIntegration(r.Context(), string(source))
	if err != nil && !errors.Is(err, storage.ErrIntegrationNotFound) {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the integration record")
		return
	}

	response, err := toIntegration(source, record)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the integration record")
		return
	}

	w.Header().Set("ETag", integrationETag(source, record))
	w.Header().Set("Cache-Control", integrationCacheControl)
	if ifNoneMatch(r.Header.Get("If-None-Match"), integrationETag(source, record)) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// RefreshIntegration forces an immediate poll of a source and returns the fresh
// record. It is an authed operator/debug gesture; unlike the public read it
// surfaces a fetch failure as an error, since the operator wants to know the
// poll did not land.
func (h *handlers) RefreshIntegration(w http.ResponseWriter, r *http.Request, source api.Source) {
	record, err := h.poller.Poll(r.Context(), string(source))
	if err != nil {
		writeProblem(w, http.StatusBadGateway, "integration_unreachable", "could not poll the source")
		return
	}

	response, err := toIntegration(source, record)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the integration record")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// toIntegration renders a stored row on the wire. A zero record (the source has
// never been polled) becomes the empty-but-present shape: the source echoed
// back, a zero fetch timestamp, and null data.
func toIntegration(source api.Source, record storage.Integration) (api.Integration, error) {
	response := api.Integration{
		Source:    source,
		FetchedAt: record.FetchedAt.UTC(),
	}
	if record.Data == "" {
		return response, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(record.Data), &data); err != nil {
		return api.Integration{}, err
	}
	response.Data = &data
	return response, nil
}

// integrationETag derives the read's validator from the source and the fetch
// timestamp — the one field that moves only when a new state was appended.
func integrationETag(source api.Source, record storage.Integration) string {
	return makeETag(string(source), record.FetchedAt.Format(time.RFC3339Nano))
}
