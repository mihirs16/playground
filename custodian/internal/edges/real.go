package edges

import (
	"context"
	"fmt"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
)

// Real constructs the production edges from config. The concrete backends — the
// AWS SDK v2 client against the instance-profile credential chain, the
// Steam/GitHub HTTP clients, and the OTel/OTLP exporter — are stubbed to return
// a clear not-implemented error until each is wired.
func Real(cfg config.Config) Set {
	return Set{
		ObjectStore:  &s3Store{bucket: cfg.MediaBucket},
		SourceClient: &httpSourceClient{},
		Telemetry:    &otlpTelemetry{endpoint: cfg.OTLPEndpoint, token: cfg.OTLPToken},
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

type otlpTelemetry struct {
	endpoint string
	token    string
}

func (t *otlpTelemetry) RecordHealth(context.Context, bool) {}

func (t *otlpTelemetry) Shutdown(context.Context) error { return nil }

type errNotImplemented struct{ what string }

func (e errNotImplemented) Error() string {
	return fmt.Sprintf("%s: not implemented", e.what)
}
