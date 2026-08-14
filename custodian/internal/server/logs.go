package server

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// logsCacheControl is the per-type caching policy for logs: cache for an hour,
// then keep serving the stale copy while revalidating in the background. Reads
// pair it with an ETag so a revalidation costs a 304, not a full body.
const logsCacheControl = "public, max-age=3600, stale-while-revalidate=86400"

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListPublicLogs serves the public blog index: listed logs only, bodies
// omitted, paged and optionally filtered by tag, with total for pagination.
func (h *handlers) ListPublicLogs(w http.ResponseWriter, r *http.Request, params api.ListPublicLogsParams) {
	query := storage.LogQuery{
		State:  string(api.Listed),
		Limit:  clampLimit(params.Limit),
		Offset: clampOffset(params.Offset),
	}
	if params.Tag != nil {
		query.Tag = *params.Tag
	}

	total, logs, err := h.db.ListLogs(r.Context(), query)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the log index")
		return
	}

	index := api.LogIndex{Total: total, Items: make([]api.LogSummary, 0, len(logs))}
	for _, log := range logs {
		index.Items = append(index.Items, toSummary(log))
	}

	if serveConditional(w, r, indexETag(query, index)) {
		return
	}
	writeJSON(w, http.StatusOK, index)
}

// GetPublicLog serves a single log by slug in any state, body included, so an
// unlisted draft is previewable at its real URL.
func (h *handlers) GetPublicLog(w http.ResponseWriter, r *http.Request, slug api.Slug) {
	log, err := h.db.GetLog(r.Context(), slug)
	if errors.Is(err, storage.ErrLogNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the log")
		return
	}

	if serveConditional(w, r, logETag(log)) {
		return
	}
	writeJSON(w, http.StatusOK, toLog(log))
}

// serveConditional sets the ETag and caching headers every log read carries,
// then honours If-None-Match: on a match it writes 304 with an empty body and
// reports true, so the caller stops. Otherwise it reports false.
func serveConditional(w http.ResponseWriter, r *http.Request, etag string) bool {
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", logsCacheControl)
	if ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// ifNoneMatch reports whether the client's If-None-Match header matches etag,
// accepting the wildcard and a comma-separated list of candidates.
func ifNoneMatch(header, etag string) bool {
	if header == "" {
		return false
	}
	if strings.TrimSpace(header) == "*" {
		return true
	}
	for _, candidate := range strings.Split(header, ",") {
		if strings.TrimSpace(candidate) == etag {
			return true
		}
	}
	return false
}

// logETag derives a detail read's validator from the log's updated_at — the one
// field that changes on every edit.
func logETag(log storage.Log) string {
	return makeETag(log.Slug, log.UpdatedAt.Format(time.RFC3339Nano))
}

// indexETag derives the index's validator from the page's contents, so any
// change to the page — a re-order, an edit, a different window — moves it.
func indexETag(query storage.LogQuery, index api.LogIndex) string {
	parts := []string{
		query.Tag,
		strconv.Itoa(query.Limit),
		strconv.Itoa(query.Offset),
		strconv.Itoa(index.Total),
	}
	for _, item := range index.Items {
		parts = append(parts, item.Slug, item.UpdatedAt.Format(time.RFC3339Nano))
	}
	return makeETag(parts...)
}

func makeETag(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf(`"%x"`, h.Sum(nil)[:16])
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func clampLimit(limit *int) int {
	if limit == nil {
		return defaultLimit
	}
	switch {
	case *limit < 1:
		return 1
	case *limit > maxLimit:
		return maxLimit
	default:
		return *limit
	}
}

func clampOffset(offset *int) int {
	if offset == nil || *offset < 0 {
		return 0
	}
	return *offset
}

func toSummary(log storage.Log) api.LogSummary {
	return api.LogSummary{
		Slug:        log.Slug,
		Title:       log.Title,
		Subtitle:    log.Subtitle,
		Description: log.Description,
		CoverImage:  log.CoverImage,
		ReadingTime: log.ReadingTime,
		Tags:        tagsPtr(log.Tags),
		State:       api.LogState(log.State),
		CreatedAt:   log.CreatedAt,
		UpdatedAt:   log.UpdatedAt,
	}
}

func toLog(log storage.Log) api.Log {
	return api.Log{
		Slug:        log.Slug,
		Title:       log.Title,
		Subtitle:    log.Subtitle,
		Description: log.Description,
		CoverImage:  log.CoverImage,
		ReadingTime: log.ReadingTime,
		Tags:        tagsPtr(log.Tags),
		State:       api.LogState(log.State),
		Body:        log.Body,
		CreatedAt:   log.CreatedAt,
		UpdatedAt:   log.UpdatedAt,
	}
}

func tagsPtr(tags []string) *[]string {
	if len(tags) == 0 {
		return nil
	}
	return &tags
}
