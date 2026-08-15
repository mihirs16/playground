package edges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// The real OTLP telemetry emits the health gauge over OTLP/HTTP to the
// configured endpoint, carrying the export token as a bearer header — exercised
// here against a local sink that stands in for a live backend.
func TestOTLPTelemetryExportsGaugeWithToken(t *testing.T) {
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

	telemetry, err := buildOTLPTelemetry(context.Background(), sink.URL, "export-secret")
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
	if gotAuth != "Bearer export-secret" {
		t.Fatalf("authorization = %q, want %q", gotAuth, "Bearer export-secret")
	}
}

// An empty endpoint yields a no-op sink rather than a boot failure, so a missing
// secret surfaces as absent telemetry.
func TestOTLPTelemetryNoEndpointIsNoop(t *testing.T) {
	telemetry := newOTLPTelemetry("", "")
	if _, ok := telemetry.(noopTelemetry); !ok {
		t.Fatalf("telemetry = %T, want noopTelemetry", telemetry)
	}
	telemetry.RecordHealth(context.Background(), false)
	if err := telemetry.Shutdown(context.Background()); err != nil {
		t.Fatalf("noop shutdown: %v", err)
	}
}
