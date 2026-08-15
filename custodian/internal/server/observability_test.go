package server_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/server"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// The poll loop is the timer that drives the health gauge: a startup pass emits
// a value through the telemetry sink without any /healthz request.
func TestPollLoopEmitsHealthGauge(t *testing.T) {
	h := newHarness(t)

	h.srv.Poller().Startup(context.Background(), discardLogger())

	value, recorded := h.edges.Telemetry.(*edges.FakeTelemetry).LastHealth()
	if !recorded {
		t.Fatal("poll loop recorded no health gauge value")
	}
	if !value {
		t.Fatal("health gauge = degraded, want healthy")
	}
}

// A third party being unreachable must never turn custodian red: an errored
// source client leaves the gauge healthy, since reachability is not a health
// input at all.
func TestThirdPartyFailureDoesNotFlipGauge(t *testing.T) {
	h := newHarness(t)
	h.sourceClient(t).SetError("steam", errors.New("steam is down"))
	h.sourceClient(t).SetError("github", errors.New("github is down"))

	h.srv.Poller().Startup(context.Background(), discardLogger())

	value, recorded := h.edges.Telemetry.(*edges.FakeTelemetry).LastHealth()
	if !recorded {
		t.Fatal("poll loop recorded no health gauge value")
	}
	if !value {
		t.Fatal("health gauge = degraded, want healthy despite unreachable sources")
	}
}

// Every /admin/* access is logged with a timestamp, the real client IP taken
// from CloudFront's X-Forwarded-For, the path, and the result — so an admin call
// the author did not make is a visible leak signal.
func TestAdminAccessIsLogged(t *testing.T) {
	var buf bytes.Buffer
	srv := newServerWithLogger(t, slog.New(slog.NewJSONHandler(&buf, nil)))
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin/v1/logs", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 70.41.3.18")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	resp.Body.Close()

	entry := findLogEntry(t, &buf, "admin access")
	if entry["time"] == nil || entry["time"] == "" {
		t.Fatal("admin access log has no timestamp")
	}
	if entry["client_ip"] != "203.0.113.7" {
		t.Fatalf("client_ip = %v, want 203.0.113.7 (leftmost XFF entry)", entry["client_ip"])
	}
	if entry["path"] != "/admin/v1/logs" {
		t.Fatalf("path = %v, want /admin/v1/logs", entry["path"])
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Fatalf("status = %v, want 200", entry["status"])
	}
}

func newServerWithLogger(t *testing.T, logger *slog.Logger) *server.Server {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "custodian.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	sum := sha256.Sum256([]byte(testAdminToken))
	cfg := config.Config{
		AdminTokenHash:  hex.EncodeToString(sum[:]),
		CORSAllowlist:   []string{testOrigin},
		MediaCDNBase:    testCDNBase,
		IntegrationKeys: map[string]string{"steam": "steam-key", "github": "github-pat"},
		PollIntervals:   map[string]time.Duration{"steam": time.Minute, "github": time.Minute},
	}
	return server.New(cfg, db, edges.NewFakes(), logger)
}

// findLogEntry returns the first JSON log line whose message matches msg.
func findLogEntry(t *testing.T, buf *bytes.Buffer, msg string) map[string]any {
	t.Helper()
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("decode log line %q: %v", line, err)
		}
		if entry["msg"] == msg {
			return entry
		}
	}
	t.Fatalf("no log entry with msg %q in:\n%s", msg, buf.String())
	return nil
}
