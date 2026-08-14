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

	// mediaBaseURL is the CDN origin a reserved media record's public url is
	// built under; see config.MediaCDNBase.
	mediaBaseURL string
}

var _ api.ServerInterface = (*handlers)(nil)

// Public surface. ListPublicLogs and GetPublicLog live in logs.go.

func (h *handlers) GetPublicIntegration(w http.ResponseWriter, _ *http.Request, _ api.Source) {
	writeNotImplemented(w)
}

func (h *handlers) GetPublicProfile(w http.ResponseWriter, _ *http.Request, _ api.ProfileKey) {
	writeNotImplemented(w)
}

// Admin surface. The log lifecycle handlers — ListAdminLogs, CreateLog,
// PatchLog, DeleteLog — live in admin_logs.go.

// The media reserve → confirm handlers — ListMedia, ReserveMedia, GetMedia,
// DeleteMedia, ConfirmMedia — live in media.go.

func (h *handlers) PutProfile(w http.ResponseWriter, _ *http.Request, _ api.ProfileKey) {
	writeNotImplemented(w)
}

func (h *handlers) RefreshIntegration(w http.ResponseWriter, _ *http.Request, _ api.Source) {
	writeNotImplemented(w)
}
