package edges

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
)

// Real constructs the production edges from config. The Steam/GitHub HTTP clients
// remain stubbed to return a clear not-implemented error until wired; the S3
// object store and the OTel/OTLP exporter are real.
//
// Each real edge that can fail to build follows the same rule: the box still
// boots, and the failure shows up loudly in the startup log rather than being
// indistinguishable from "not configured". A telemetry exporter that fails drops
// to the no-op sink; an object store that fails to build becomes one that returns
// the build error on every call, so media presigning and the bucket health check
// fail visibly instead of panicking.
func Real(cfg config.Config, logger *slog.Logger) Set {
	telemetry, err := newOTLPTelemetry(cfg.OTLPEndpoint, cfg.OTLPAuthorization, logger)
	if err != nil {
		logger.Error("OTLP telemetry is configured but its exporter failed to build; running with telemetry disabled",
			"endpoint", cfg.OTLPEndpoint, "error", err)
	}

	var objectStore ObjectStore
	store, err := newS3Store(context.Background(), cfg.MediaBucket)
	if err != nil {
		logger.Error("S3 object store failed to build; media uploads and the bucket health check will error",
			"bucket", cfg.MediaBucket, "error", err)
		objectStore = errObjectStore{err}
	} else {
		objectStore = store
	}

	return Set{
		ObjectStore:  objectStore,
		SourceClient: &httpSourceClient{},
		Telemetry:    telemetry,
	}
}

// errObjectStore stands in when the real S3 store cannot be built, returning the
// build failure from every call so the fault is visible at the edge rather than a
// nil-pointer panic deeper in.
type errObjectStore struct{ err error }

func (e errObjectStore) PresignPut(context.Context, string, string, time.Duration) (string, error) {
	return "", e.err
}
func (e errObjectStore) HeadObject(context.Context, string) (bool, error) { return false, e.err }
func (e errObjectStore) HeadBucket(context.Context) error                 { return e.err }

type httpSourceClient struct{}

func (c *httpSourceClient) Fetch(context.Context, string, string, string) (FetchResult, error) {
	return FetchResult{}, errNotImplemented{"source client fetch"}
}

type errNotImplemented struct{ what string }

func (e errNotImplemented) Error() string {
	return fmt.Sprintf("%s: not implemented", e.what)
}
