package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/mihirs16/playground/custodian/internal/edges"
)

// integrationRowCount reads how many timeseries rows a source has, so a test can
// assert append-on-change: a changed poll adds a row, an idle one adds nothing.
func (h *harness) integrationRowCount(t *testing.T, source string) int {
	t.Helper()
	var count int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM integration WHERE source = ?`, source).Scan(&count); err != nil {
		t.Fatalf("count integration rows: %v", err)
	}
	return count
}

// TestRefreshAppendsOnChange proves a poll that observes a new state appends a
// row and returns it fresh.
func TestRefreshAppendsOnChange(t *testing.T) {
	h := newHarness(t)
	h.sourceClient(t).SetResult("steam", edges.FetchResult{
		ETag: `"v1"`,
		Data: map[string]any{"now_playing": "Factorio"},
	})

	resp := h.request(t, http.MethodPost, "/admin/v1/integrations/steam/refresh", adminAuth())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var record struct {
		Source    string         `json:"source"`
		FetchedAt string         `json:"fetched_at"`
		Data      map[string]any `json:"data"`
	}
	decode(t, resp, &record)
	if record.Source != "steam" {
		t.Fatalf("source = %q, want steam", record.Source)
	}
	if record.Data["now_playing"] != "Factorio" {
		t.Fatalf("data = %v, want the polled state", record.Data)
	}
	if record.FetchedAt == "" {
		t.Fatal("fetched_at is empty; a changed poll must stamp it")
	}
	if n := h.integrationRowCount(t, "steam"); n != 1 {
		t.Fatalf("row count = %d, want 1", n)
	}
}

// TestIdlePollAppendsNothing proves an unchanged body inserts no row and does
// not move the fetch timestamp — idle polls are invisible in the timeseries.
func TestIdlePollAppendsNothing(t *testing.T) {
	h := newHarness(t)
	h.sourceClient(t).SetResult("steam", edges.FetchResult{
		ETag: `"v1"`,
		Data: map[string]any{"now_playing": "Factorio"},
	})

	first, err := h.srv.Poller().Poll(context.Background(), "steam")
	if err != nil {
		t.Fatalf("first poll: %v", err)
	}

	// Same state again: nothing changed.
	second, err := h.srv.Poller().Poll(context.Background(), "steam")
	if err != nil {
		t.Fatalf("second poll: %v", err)
	}

	if n := h.integrationRowCount(t, "steam"); n != 1 {
		t.Fatalf("row count = %d, want 1 (idle poll must not append)", n)
	}
	if !second.FetchedAt.Equal(first.FetchedAt) {
		t.Fatalf("fetched_at moved on an idle poll: %v -> %v", first.FetchedAt, second.FetchedAt)
	}
}

// TestNotModifiedKeepsLastRow proves a 304 leaves the timeseries untouched and
// serves the last stored state as last-known-good.
func TestNotModifiedKeepsLastRow(t *testing.T) {
	h := newHarness(t)
	client := h.sourceClient(t)
	client.SetResult("github", edges.FetchResult{
		ETag: `"abc"`,
		Data: map[string]any{"public_repos": float64(42)},
	})
	if _, err := h.srv.Poller().Poll(context.Background(), "github"); err != nil {
		t.Fatalf("prime poll: %v", err)
	}

	client.SetResult("github", edges.FetchResult{NotModified: true})
	if _, err := h.srv.Poller().Poll(context.Background(), "github"); err != nil {
		t.Fatalf("304 poll: %v", err)
	}

	if n := h.integrationRowCount(t, "github"); n != 1 {
		t.Fatalf("row count = %d, want 1 (304 must not append)", n)
	}

	resp := h.request(t, http.MethodGet, "/v1/integrations/github", nil)
	defer resp.Body.Close()
	var record struct {
		Data map[string]any `json:"data"`
	}
	decode(t, resp, &record)
	if record.Data["public_repos"] != float64(42) {
		t.Fatalf("data = %v, want the last-known-good state", record.Data)
	}
}

// TestUnreachableSourceServesLastKnownGood proves a network failure never flips
// the stored state: refresh reports the failure, but the read still serves the
// prior row with its original timestamp — indistinguishable from an idle poll.
func TestUnreachableSourceServesLastKnownGood(t *testing.T) {
	h := newHarness(t)
	client := h.sourceClient(t)
	client.SetResult("steam", edges.FetchResult{
		ETag: `"v1"`,
		Data: map[string]any{"now_playing": "Factorio"},
	})
	primed, err := h.srv.Poller().Poll(context.Background(), "steam")
	if err != nil {
		t.Fatalf("prime poll: %v", err)
	}

	client.SetError("steam", context.DeadlineExceeded)

	refresh := h.request(t, http.MethodPost, "/admin/v1/integrations/steam/refresh", adminAuth())
	defer refresh.Body.Close()
	if refresh.StatusCode != http.StatusBadGateway {
		t.Fatalf("refresh status = %d, want 502 for an unreachable source", refresh.StatusCode)
	}

	if n := h.integrationRowCount(t, "steam"); n != 1 {
		t.Fatalf("row count = %d, want 1 (unreachable must not append)", n)
	}

	read := h.request(t, http.MethodGet, "/v1/integrations/steam", nil)
	defer read.Body.Close()
	var record struct {
		FetchedAt time.Time      `json:"fetched_at"`
		Data      map[string]any `json:"data"`
	}
	decode(t, read, &record)
	if record.Data["now_playing"] != "Factorio" {
		t.Fatalf("data = %v, want last-known-good", record.Data)
	}
	if !record.FetchedAt.Equal(primed.FetchedAt.UTC()) {
		t.Fatalf("fetched_at = %v, want unchanged %v", record.FetchedAt, primed.FetchedAt.UTC())
	}
}

// TestStartupPollSeedsRow proves every poller runs once at startup, so a row
// exists before persona's first read.
func TestStartupPollSeedsRow(t *testing.T) {
	h := newHarness(t)
	h.sourceClient(t).SetResult("steam", edges.FetchResult{
		Data: map[string]any{"now_playing": "Stardew Valley"},
	})

	h.srv.Poller().Startup(context.Background(), discardLogger())

	if n := h.integrationRowCount(t, "steam"); n != 1 {
		t.Fatalf("row count = %d, want 1 after startup", n)
	}
}

// TestReadEmptyButPresentBeforeFirstPoll proves a source never polled returns a
// legal empty-but-present shape, not a 404: the source echoed back with null
// data and a present (zero) timestamp.
func TestReadEmptyButPresentBeforeFirstPoll(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/v1/integrations/steam", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (empty-but-present, never 404)", resp.StatusCode)
	}
	var record struct {
		Source    string         `json:"source"`
		FetchedAt string         `json:"fetched_at"`
		Data      map[string]any `json:"data"`
	}
	decode(t, resp, &record)
	if record.Source != "steam" {
		t.Fatalf("source = %q, want steam", record.Source)
	}
	if record.FetchedAt == "" {
		t.Fatal("fetched_at is absent; the shape must be present even when empty")
	}
	if record.Data != nil {
		t.Fatalf("data = %v, want null before any poll", record.Data)
	}
}

// TestReadCarriesShortTTLAndRevalidates proves the read surface carries the
// short TTL and an ETag, and honours If-None-Match with a 304.
func TestReadCarriesShortTTLAndRevalidates(t *testing.T) {
	h := newHarness(t)
	h.sourceClient(t).SetResult("github", edges.FetchResult{
		Data: map[string]any{"public_repos": float64(7)},
	})
	if _, err := h.srv.Poller().Poll(context.Background(), "github"); err != nil {
		t.Fatalf("poll: %v", err)
	}

	first := h.request(t, http.MethodGet, "/v1/integrations/github", nil)
	defer first.Body.Close()
	if cc := first.Header.Get("Cache-Control"); cc != "public, max-age=60, s-maxage=60, stale-while-revalidate=300" {
		t.Fatalf("cache-control = %q, want the short integration TTL", cc)
	}
	etag := first.Header.Get("ETag")
	if etag == "" {
		t.Fatal("ETag is empty; the read cannot be revalidated")
	}

	second := h.request(t, http.MethodGet, "/v1/integrations/github", map[string]string{"If-None-Match": etag})
	defer second.Body.Close()
	if second.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 on a matching If-None-Match", second.StatusCode)
	}
}

// TestReapDeletesExpiredPending proves the poll loop's reap removes a pending
// reservation past its window while leaving a fresh one and an available record.
func TestReapDeletesExpiredPending(t *testing.T) {
	h := newHarness(t)
	now := time.Now().UTC()

	h.seedMedia(t, "stale-pending", "pending", now.Add(-time.Hour))
	h.seedMedia(t, "fresh-pending", "pending", now.Add(time.Hour))
	h.seedMedia(t, "already-available", "available", now.Add(-time.Hour))

	reaped, err := h.srv.Poller().Reap(context.Background(), now)
	if err != nil {
		t.Fatalf("reap: %v", err)
	}
	if reaped != 1 {
		t.Fatalf("reaped = %d, want 1", reaped)
	}

	if _, ok := h.mediaState(t, "stale-pending"); ok {
		t.Fatal("stale pending reservation survived the reap")
	}
	if _, ok := h.mediaState(t, "fresh-pending"); !ok {
		t.Fatal("fresh reservation was wrongly reaped")
	}
	if _, ok := h.mediaState(t, "already-available"); !ok {
		t.Fatal("available record was wrongly reaped")
	}
}
