package edges

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

// The real OTLP telemetry emits the health gauge over OTLP/HTTP to the
// configured endpoint, carrying the configured Authorization header verbatim —
// exercised here against a local sink that stands in for a live backend.
func TestOTLPTelemetryExportsGaugeWithAuthorizationHeader(t *testing.T) {
	var (
		mu      sync.Mutex
		hits    int
		gotAuth string
	)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()

	const authorization = "Basic MTIzNDU2OmdsYy1yZWFsLXRva2Vu"

	telemetry, err := buildOTLPTelemetry(context.Background(), sink.URL, authorization, discardLogger())
	if err != nil {
		t.Fatalf("build otlp telemetry: %v", err)
	}

	telemetry.RecordHealth(context.Background(), true)

	// Shutdown flushes the periodic reader, forcing the pending gauge export.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := telemetry.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if hits == 0 {
		t.Fatal("sink received no OTLP export")
	}
	if gotAuth != authorization {
		t.Fatalf("authorization = %q, want %q (sent verbatim)", gotAuth, authorization)
	}
}

// An empty endpoint yields a no-op sink and no error: telemetry is deliberately
// off, so a missing secret surfaces as absent telemetry, not a boot failure.
func TestOTLPTelemetryNoEndpointIsNoop(t *testing.T) {
	telemetry, err := newOTLPTelemetry("", "", discardLogger())
	if err != nil {
		t.Fatalf("empty endpoint returned error: %v", err)
	}
	if _, ok := telemetry.(noopTelemetry); !ok {
		t.Fatalf("telemetry = %T, want noopTelemetry", telemetry)
	}
	telemetry.RecordHealth(context.Background(), false)
	if err := telemetry.Shutdown(context.Background()); err != nil {
		t.Fatalf("noop shutdown: %v", err)
	}
}

// A misauthenticated export — the 401 a real Grafana Cloud stack returns for a
// bad credential — must not be silently swallowed. The OTel SDK reports export
// failures on its periodic loop through the process error handler; custodian
// installs one that routes them to its own logger, so a configured-but-rejected
// exporter is loud and distinguishable from a quiet no-op sink, not buried in
// the SDK's internal log.
func TestOTLPTelemetryExportErrorsRouteToLogger(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError}))

	telemetry, err := buildOTLPTelemetry(context.Background(), "http://localhost:0", "Basic wrong-credential", logger)
	if err != nil {
		t.Fatalf("build otlp telemetry: %v", err)
	}
	defer telemetry.Shutdown(context.Background())

	// Stand in for a periodic-export failure (the shape a Grafana Cloud 401 takes
	// once the providers are up): the SDK hands it to the process error handler.
	otel.Handle(errors.New("401 Unauthorized"))

	if !strings.Contains(logs.String(), "OTLP export failed") {
		t.Fatalf("export error was not surfaced through custodian's logger; logs = %q", logs.String())
	}
}

// signalURL appends the per-signal path to the base endpoint the OTEL way, and
// tolerates a trailing slash on the base — so Grafana Cloud's ".../otlp" gateway
// is reached at ".../otlp/v1/metrics" whether or not the operator adds the slash.
func TestSignalURL(t *testing.T) {
	cases := []struct {
		endpoint, signal, want string
	}{
		{"https://otlp-gateway-prod-gb-south-1.grafana.net/otlp", "metrics", "https://otlp-gateway-prod-gb-south-1.grafana.net/otlp/v1/metrics"},
		{"https://otlp-gateway-prod-gb-south-1.grafana.net/otlp/", "traces", "https://otlp-gateway-prod-gb-south-1.grafana.net/otlp/v1/traces"},
		{"http://localhost:4318", "logs", "http://localhost:4318/v1/logs"},
	}
	for _, c := range cases {
		if got := signalURL(c.endpoint, c.signal); got != c.want {
			t.Errorf("signalURL(%q, %q) = %q, want %q", c.endpoint, c.signal, got, c.want)
		}
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
