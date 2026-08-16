package edges

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
)

// Real constructs the production edges from config. The concrete backends — the
// AWS SDK v2 client against the instance-profile credential chain, the
// Steam/GitHub HTTP clients, and the OTel/OTLP exporter — are stubbed to return
// a clear not-implemented error until each is wired.
//
// A telemetry exporter that is configured (endpoint set) but cannot be built is
// logged loudly and then dropped to the no-op sink: the box still boots, but the
// misconfiguration shows up in the startup log rather than being indistinguishable
// from "no telemetry configured".
func Real(cfg config.Config, logger *slog.Logger) Set {
	telemetry, err := newOTLPTelemetry(cfg.OTLPEndpoint, cfg.OTLPAuthorization, logger)
	if err != nil {
		logger.Error("OTLP telemetry is configured but its exporter failed to build; running with telemetry disabled",
			"endpoint", cfg.OTLPEndpoint, "error", err)
	}
	return Set{
		ObjectStore:  &s3Store{bucket: cfg.MediaBucket},
		SourceClient: &httpSourceClient{},
		Telemetry:    telemetry,
	}
}

type s3Store struct {
	bucket string
}

func (s *s3Store) PresignPut(context.Context, string, string, time.Duration) (string, error) {
	return "", errNotImplemented{"s3 presign"}
}

func (s *s3Store) HeadObject(context.Context, string) (bool, error) {
	return false, errNotImplemented{"s3 head object"}
}

func (s *s3Store) HeadBucket(context.Context) error {
	return errNotImplemented{"s3 head bucket"}
}

type httpSourceClient struct{}

func (c *httpSourceClient) Fetch(context.Context, string, string, string) (FetchResult, error) {
	return FetchResult{}, errNotImplemented{"source client fetch"}
}

type errNotImplemented struct{ what string }

func (e errNotImplemented) Error() string {
	return fmt.Sprintf("%s: not implemented", e.what)
}
