package server

import (
	"context"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// integrationPoller is the poll-one-source capability the integration handlers
// reach for: the read surface reads stored rows directly, but the refresh
// gesture forces an immediate poll.
type integrationPoller interface {
	Poll(ctx context.Context, source string) (storage.Integration, error)
}

// handlers implements the generated api.ServerInterface over the real database,
// edges, and the background poller.
type handlers struct {
	db     *storage.DB
	edges  edges.Set
	poller integrationPoller

	// mediaBaseURL is the CDN origin a reserved media record's public url is
	// built under; see config.MediaCDNBase.
	mediaBaseURL string
}

var _ api.ServerInterface = (*handlers)(nil)

// Public surface. ListPublicLogs and GetPublicLog live in logs.go; the profile
// read GetPublicProfile lives in profile.go; the integration read
// GetPublicIntegration lives in integration.go.

// Admin surface. The log lifecycle handlers — ListAdminLogs, CreateLog,
// PatchLog, DeleteLog — live in admin_logs.go. The media reserve → confirm
// handlers — ListMedia, ReserveMedia, GetMedia, DeleteMedia, ConfirmMedia — live
// in media.go. The profile write PutProfile lives in profile.go. The integration
// RefreshIntegration lives in integration.go.
