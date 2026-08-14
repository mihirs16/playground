package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func day(n int) time.Time {
	return time.Date(2026, time.January, n, 12, 0, 0, 0, time.UTC)
}

// TestListPublicLogsReturnsListedSummaries covers the index contract: total and
// items, listed-only, and no body on a summary.
func TestListPublicLogsReturnsListedSummaries(t *testing.T) {
	h := newHarness(t)
	h.seed(t,
		seedLog{slug: "shipped", title: "Shipped", body: "full body here", state: "listed", createdAt: day(2), updatedAt: day(2)},
		seedLog{slug: "draft", title: "Draft", body: "secret", state: "unlisted", createdAt: day(3), updatedAt: day(3)},
	)

	resp := h.request(t, http.MethodGet, "/v1/logs", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var index struct {
		Total int              `json:"total"`
		Items []map[string]any `json:"items"`
	}
	decode(t, resp, &index)

	if index.Total != 1 {
		t.Fatalf("total = %d, want 1 (listed only)", index.Total)
	}
	if len(index.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(index.Items))
	}
	item := index.Items[0]
	if item["slug"] != "shipped" {
		t.Fatalf("slug = %v, want shipped", item["slug"])
	}
	if _, present := item["body"]; present {
		t.Fatalf("summary must omit body, got %v", item["body"])
	}
}

func TestListPublicLogsPagesAndFiltersByTag(t *testing.T) {
	h := newHarness(t)
	h.seed(t,
		seedLog{slug: "go-1", title: "Go one", state: "listed", tags: []string{"go"}, createdAt: day(1), updatedAt: day(1)},
		seedLog{slug: "go-2", title: "Go two", state: "listed", tags: []string{"go", "sqlite"}, createdAt: day(2), updatedAt: day(2)},
		seedLog{slug: "rust-1", title: "Rust one", state: "listed", tags: []string{"rust"}, createdAt: day(3), updatedAt: day(3)},
	)

	// Newest first, one per page.
	resp := h.request(t, http.MethodGet, "/v1/logs?limit=1&offset=1", nil)
	defer resp.Body.Close()
	var page struct {
		Total int `json:"total"`
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}
	decode(t, resp, &page)
	if page.Total != 3 {
		t.Fatalf("total = %d, want 3", page.Total)
	}
	if len(page.Items) != 1 || page.Items[0].Slug != "go-2" {
		t.Fatalf("page = %+v, want single go-2 (second newest)", page.Items)
	}

	// Tag filter narrows both total and items.
	tagResp := h.request(t, http.MethodGet, "/v1/logs?tag=go", nil)
	defer tagResp.Body.Close()
	var tagged struct {
		Total int `json:"total"`
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}
	decode(t, tagResp, &tagged)
	if tagged.Total != 2 || len(tagged.Items) != 2 {
		t.Fatalf("tag=go total/items = %d/%d, want 2/2", tagged.Total, len(tagged.Items))
	}
}

func TestGetPublicLogReturnsFullBodyForUnlisted(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "draft", title: "Draft", body: "# hidden draft", state: "unlisted", createdAt: day(1), updatedAt: day(1)})

	resp := h.request(t, http.MethodGet, "/v1/logs/draft", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var log struct {
		Slug  string `json:"slug"`
		Body  string `json:"body"`
		State string `json:"state"`
	}
	decode(t, resp, &log)
	if log.Body != "# hidden draft" {
		t.Fatalf("body = %q, want the full draft body", log.Body)
	}
	if log.State != "unlisted" {
		t.Fatalf("state = %q, want unlisted", log.State)
	}
}

func TestGetPublicLogUnknownSlugIs404(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/v1/logs/nope", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
}

func TestReadsCarryETagAndCacheControl(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "shipped", title: "Shipped", state: "listed", createdAt: day(1), updatedAt: day(1)})

	for _, path := range []string{"/v1/logs", "/v1/logs/shipped"} {
		resp := h.request(t, http.MethodGet, path, nil)
		resp.Body.Close()
		if etag := resp.Header.Get("ETag"); etag == "" {
			t.Fatalf("%s: missing ETag", path)
		}
		if cc := resp.Header.Get("Cache-Control"); cc == "" {
			t.Fatalf("%s: missing Cache-Control", path)
		}
	}
}

func TestIfNoneMatchReturns304WithEmptyBody(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "shipped", title: "Shipped", body: "body", state: "listed", createdAt: day(1), updatedAt: day(1)})

	for _, path := range []string{"/v1/logs", "/v1/logs/shipped"} {
		first := h.request(t, http.MethodGet, path, nil)
		etag := first.Header.Get("ETag")
		first.Body.Close()

		second := h.request(t, http.MethodGet, path, map[string]string{"If-None-Match": etag})
		body, _ := io.ReadAll(second.Body)
		second.Body.Close()

		if second.StatusCode != http.StatusNotModified {
			t.Fatalf("%s: status = %d, want 304", path, second.StatusCode)
		}
		if len(body) != 0 {
			t.Fatalf("%s: 304 body = %q, want empty", path, body)
		}
		if second.Header.Get("ETag") != etag {
			t.Fatalf("%s: 304 must still carry the ETag", path)
		}
	}
}

// TestReadsDoNotMutateStoredState asserts the read surface leaves the database
// exactly as it found it — an unlisted draft stays unlisted after a preview.
func TestReadsDoNotMutateStoredState(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "draft", title: "Draft", state: "unlisted", createdAt: day(1), updatedAt: day(1)})

	h.request(t, http.MethodGet, "/v1/logs/draft", nil).Body.Close()
	h.request(t, http.MethodGet, "/v1/logs", nil).Body.Close()

	state, ok := h.logState(t, "draft")
	if !ok {
		t.Fatal("draft vanished from the database")
	}
	if state != "unlisted" {
		t.Fatalf("state = %q, want unlisted (reads must not mutate)", state)
	}
}

func decode(t *testing.T, resp *http.Response, into any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
