package server

import (
	"net/http"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// handlers implements the generated api.ServerInterface over the real database
// and edges. Each method answers not-implemented until its surface's behaviour
// is wired.
type handlers struct {
	db    *storage.DB
	edges edges.Set
}

var _ api.ServerInterface = (*handlers)(nil)

// Public surface.

func (h *handlers) ListPublicLogs(w http.ResponseWriter, _ *http.Request, _ api.ListPublicLogsParams) {
	writeNotImplemented(w)
}

func (h *handlers) GetPublicLog(w http.ResponseWriter, _ *http.Request, _ api.Slug) {
	writeNotImplemented(w)
}

func (h *handlers) GetPublicIntegration(w http.ResponseWriter, _ *http.Request, _ api.Source) {
	writeNotImplemented(w)
}

func (h *handlers) GetPublicProfile(w http.ResponseWriter, _ *http.Request, _ api.ProfileKey) {
	writeNotImplemented(w)
}

// Admin surface.

func (h *handlers) ListAdminLogs(w http.ResponseWriter, _ *http.Request, _ api.ListAdminLogsParams) {
	writeNotImplemented(w)
}

func (h *handlers) CreateLog(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func (h *handlers) PatchLog(w http.ResponseWriter, _ *http.Request, _ api.Slug) {
	writeNotImplemented(w)
}

func (h *handlers) DeleteLog(w http.ResponseWriter, _ *http.Request, _ api.Slug) {
	writeNotImplemented(w)
}

func (h *handlers) ListMedia(w http.ResponseWriter, _ *http.Request, _ api.ListMediaParams) {
	writeNotImplemented(w)
}

func (h *handlers) ReserveMedia(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func (h *handlers) GetMedia(w http.ResponseWriter, _ *http.Request, _ api.MediaKey) {
	writeNotImplemented(w)
}

func (h *handlers) DeleteMedia(w http.ResponseWriter, _ *http.Request, _ api.MediaKey) {
	writeNotImplemented(w)
}

func (h *handlers) ConfirmMedia(w http.ResponseWriter, _ *http.Request, _ api.MediaKey) {
	writeNotImplemented(w)
}

func (h *handlers) PutProfile(w http.ResponseWriter, _ *http.Request, _ api.ProfileKey) {
	writeNotImplemented(w)
}

func (h *handlers) RefreshIntegration(w http.ResponseWriter, _ *http.Request, _ api.Source) {
	writeNotImplemented(w)
}

func (h *handlers) PutIntegrationCredential(w http.ResponseWriter, _ *http.Request, _ api.Source) {
	writeNotImplemented(w)
}
