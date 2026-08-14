package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mihirs16/playground/custodian/internal/apiclient"
	"github.com/mihirs16/playground/custodian/internal/edges"
)

func TestHealthzReportsHealthy(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/healthz", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	value, recorded := h.edges.Telemetry.(*edges.FakeTelemetry).LastHealth()
	if !recorded {
		t.Fatal("no health gauge value recorded")
	}
	if !value {
		t.Fatal("health gauge = degraded, want healthy")
	}
}

func TestHealthzDegradedWhenBucketUnreachable(t *testing.T) {
	h := newHarness(t)
	h.edges.ObjectStore.(*edges.FakeObjectStore).BucketErr = context.DeadlineExceeded

	resp := h.request(t, http.MethodGet, "/healthz", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

// A public route whose behaviour is not yet wired still reaches its handler
// through real routing and returns a problem+json placeholder — proof the whole
// stack is wired end to end.
func TestPublicRouteReachesHandler(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/v1/integrations/steam", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}

	var problem struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "not_implemented" {
		t.Fatalf("code = %q, want not_implemented", problem.Code)
	}
}

func TestAdminRequiresBearer(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/admin/v1/logs", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
}

func TestAdminAcceptsValidBearer(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/admin/v1/logs", adminAuth())
	defer resp.Body.Close()

	// Past auth, the admin index answers 200 — not 401.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestAdminRejectsWrongBearer(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/admin/v1/logs", map[string]string{
		"Authorization": "Bearer not-the-token",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPublicCORSAllowsListedOrigin(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodOptions, "/v1/logs", map[string]string{
		"Origin":                        testOrigin,
		"Access-Control-Request-Method": "GET",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Fatalf("allow-origin = %q, want %q", got, testOrigin)
	}
	if exposed := resp.Header.Get("Access-Control-Expose-Headers"); exposed != "ETag" {
		t.Fatalf("expose-headers = %q, want ETag", exposed)
	}
}

func TestPublicCORSIgnoresUnlistedOrigin(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/v1/logs", map[string]string{
		"Origin": "https://evil.example.com",
	})
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q, want empty", got)
	}
}

func TestAdminSurfaceHasNoCORS(t *testing.T) {
	h := newHarness(t)

	headers := adminAuth()
	headers["Origin"] = testOrigin
	resp := h.request(t, http.MethodGet, "/admin/v1/logs", headers)
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("admin allow-origin = %q, want empty", got)
	}
}

// The generated client exercises the same published contract broom will use.
func TestGeneratedClientReachesPublicSurface(t *testing.T) {
	h := newHarness(t)

	client, err := apiclient.NewClientWithResponses(h.server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.ListPublicLogsWithResponse(context.Background(), nil)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode())
	}
	if resp.JSON200 == nil || resp.JSON200.Total != 0 {
		t.Fatalf("expected an empty index, got %+v", resp.JSON200)
	}
}
