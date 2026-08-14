package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// slugPattern is the shape an author-chosen slug must take: lowercase
// alphanumeric words joined by single hyphens, so it drops straight into a URL.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ListAdminLogs serves the admin index: logs in any state (drafts included),
// optionally narrowed to one state, bodies omitted, paged.
func (h *handlers) ListAdminLogs(w http.ResponseWriter, r *http.Request, params api.ListAdminLogsParams) {
	query := storage.LogQuery{
		Limit:  clampLimit(params.Limit),
		Offset: clampOffset(params.Offset),
	}
	if params.Tag != nil {
		query.Tag = *params.Tag
	}
	if params.State != nil {
		query.State = string(*params.State)
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
	writeJSON(w, http.StatusOK, index)
}

// CreateLog creates a log as an unlisted draft with the author-chosen slug. A
// slug already in use is a 409 slug_conflict.
func (h *handlers) CreateLog(w http.ResponseWriter, r *http.Request) {
	var body api.LogCreate
	if !decodeJSON(w, r, &body) {
		return
	}
	if fields := validateCreate(body); len(fields) > 0 {
		writeValidationProblem(w, fields)
		return
	}

	log := storage.Log{
		Slug:        body.Slug,
		Title:       body.Title,
		Subtitle:    body.Subtitle,
		Description: body.Description,
		CoverImage:  body.CoverImage,
		Tags:        derefSlice(body.Tags),
		Body:        derefString(body.Body),
		State:       string(api.Unlisted),
	}

	created, err := h.db.CreateLog(r.Context(), log)
	if errors.Is(err, storage.ErrSlugConflict) {
		writeProblem(w, http.StatusConflict, "slug_conflict", "a log with that slug already exists")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not create the log")
		return
	}
	writeJSON(w, http.StatusCreated, toLog(created))
}

// PatchLog partially updates a log: state transitions publish and unpublish, and
// a slug rename is a server-performed move — allowed only while the log is
// unlisted, frozen once listed. Only the fields present in the body change.
func (h *handlers) PatchLog(w http.ResponseWriter, r *http.Request, slug api.Slug) {
	var body api.LogPatch
	if !decodeJSON(w, r, &body) {
		return
	}
	if fields := validatePatch(body); len(fields) > 0 {
		writeValidationProblem(w, fields)
		return
	}

	existing, err := h.db.GetLog(r.Context(), slug)
	if errors.Is(err, storage.ErrLogNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the log")
		return
	}

	renaming := body.Slug != nil && *body.Slug != existing.Slug
	if renaming && existing.State == string(api.Listed) {
		writeProblem(w, http.StatusConflict, "slug_frozen_while_listed", "a listed log's slug cannot be changed; unlist it first")
		return
	}

	updated, err := h.db.UpdateLog(r.Context(), slug, applyPatch(existing, body))
	if errors.Is(err, storage.ErrSlugConflict) {
		writeProblem(w, http.StatusConflict, "slug_conflict", "a log with that slug already exists")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not update the log")
		return
	}
	writeJSON(w, http.StatusOK, toLog(updated))
}

// DeleteLog removes a log entirely.
func (h *handlers) DeleteLog(w http.ResponseWriter, r *http.Request, slug api.Slug) {
	err := h.db.DeleteLog(r.Context(), slug)
	if errors.Is(err, storage.ErrLogNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not delete the log")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyPatch folds the present fields of a patch onto the stored log, leaving
// absent fields untouched. Slug and state are ordinary fields here; the caller
// has already enforced the rename-while-listed rule.
func applyPatch(log storage.Log, patch api.LogPatch) storage.Log {
	if patch.Slug != nil {
		log.Slug = *patch.Slug
	}
	if patch.Title != nil {
		log.Title = *patch.Title
	}
	if patch.Subtitle != nil {
		log.Subtitle = patch.Subtitle
	}
	if patch.Description != nil {
		log.Description = patch.Description
	}
	if patch.CoverImage != nil {
		log.CoverImage = patch.CoverImage
	}
	if patch.Tags != nil {
		log.Tags = *patch.Tags
	}
	if patch.Body != nil {
		log.Body = *patch.Body
	}
	if patch.State != nil {
		log.State = string(*patch.State)
	}
	return log
}

func validateCreate(body api.LogCreate) []api.FieldError {
	var fields []api.FieldError
	fields = append(fields, validateSlug(body.Slug)...)
	if body.Title == "" {
		fields = append(fields, api.FieldError{Field: "title", Message: "title is required"})
	}
	return fields
}

func validatePatch(body api.LogPatch) []api.FieldError {
	var fields []api.FieldError
	if body.Slug != nil {
		fields = append(fields, validateSlug(*body.Slug)...)
	}
	if body.Title != nil && *body.Title == "" {
		fields = append(fields, api.FieldError{Field: "title", Message: "title cannot be empty"})
	}
	if body.State != nil && *body.State != api.Listed && *body.State != api.Unlisted {
		fields = append(fields, api.FieldError{Field: "state", Message: "state must be listed or unlisted"})
	}
	return fields
}

func validateSlug(slug string) []api.FieldError {
	if slug == "" {
		return []api.FieldError{{Field: "slug", Message: "slug is required"}}
	}
	if !slugPattern.MatchString(slug) {
		return []api.FieldError{{Field: "slug", Message: "slug must be lowercase alphanumeric words joined by hyphens"}}
	}
	return nil
}

// decodeJSON fills dst from the request body, answering a 400 problem and
// reporting false when the body is not valid JSON.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return false
	}
	return true
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefSlice(s *[]string) []string {
	if s == nil {
		return nil
	}
	return *s
}
