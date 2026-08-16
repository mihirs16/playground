package edges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// The real source client is exercised the same way the S3 store and OTLP sink
// are: against a local HTTP stand-in for the live third party, with the adapter's
// base URL pointed at that stand-in so no test ever reaches the network.
func testSource(steam, github sourceAdapter) *sourceClient {
	return &sourceClient{
		http: http.DefaultClient,
		adapters: map[string]sourceAdapter{
			"steam":  steam,
			"github": github,
		},
	}
}

// Steam's nested two-week aggregate collapses to a flat games list plus the
// summed two-week playtime, with each game's icon composed into a full media URL.
// The credential and steamid ride on the query string custodian sends.
func TestSteamShapesRecentlyPlayed(t *testing.T) {
	var gotQuery url.Values
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "GetRecentlyPlayedGames") {
			t.Errorf("path = %q, want the recently-played endpoint", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		w.Write([]byte(`{"response":{"total_count":2,"games":[
			{"appid":570,"name":"Dota 2","playtime_2weeks":120,"playtime_forever":9000,"img_icon_url":"abc123"},
			{"appid":730,"name":"CS2","playtime_2weeks":30,"playtime_forever":4000,"img_icon_url":"def456"}
		]}}`))
	}))
	defer sink.Close()

	client := testSource(steamAdapter{baseURL: sink.URL, steamID: "76561197960434622"}, nil)

	result, err := client.Fetch(context.Background(), "steam", "steam-key", "")
	if err != nil {
		t.Fatalf("fetch steam: %v", err)
	}
	if result.NotModified {
		t.Fatal("steam fetch reported NotModified; Steam serves no ETag")
	}

	if got := gotQuery.Get("key"); got != "steam-key" {
		t.Errorf("key = %q, want the per-call credential", got)
	}
	if got := gotQuery.Get("steamid"); got != "76561197960434622" {
		t.Errorf("steamid = %q, want the configured id", got)
	}

	if got := result.Data["total_playtime_2weeks_minutes"]; got != 150 {
		t.Errorf("total_playtime_2weeks_minutes = %v, want 150 (120+30)", got)
	}

	games, ok := result.Data["games"].([]map[string]any)
	if !ok || len(games) != 2 {
		t.Fatalf("games = %v, want two shaped games", result.Data["games"])
	}
	if games[0]["name"] != "Dota 2" || games[0]["playtime_2weeks_minutes"] != 120 {
		t.Errorf("first game shaped wrong: %v", games[0])
	}
	wantIcon := steamMedia + "/570/abc123.jpg"
	if games[0]["icon_url"] != wantIcon {
		t.Errorf("icon_url = %v, want %q", games[0]["icon_url"], wantIcon)
	}
}

// A Steam HTTP failure is an error, not an empty result — an unreachable source
// must keep last-known-good rather than be written as an empty state change.
func TestSteamNonOKIsAnError(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer sink.Close()

	client := testSource(steamAdapter{baseURL: sink.URL, steamID: "1"}, nil)

	if _, err := client.Fetch(context.Background(), "steam", "k", ""); err == nil {
		t.Fatal("steam 500 returned nil error, want the failure surfaced")
	}
}

// GitHub's feed is filtered to authored-activity event types and shaped flat. The
// PAT rides as a bearer token, and the response ETag is returned verbatim — weak
// validator and all — so it round-trips as If-None-Match on the next poll.
func TestGitHubFiltersAndShapesFeed(t *testing.T) {
	var gotAuth, gotIfNoneMatch string
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/users/octocat/events/public") {
			t.Errorf("path = %q, want the user public-events endpoint", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		w.Header().Set("ETag", `W/"etag-value"`)
		w.Write([]byte(`[
			{"id":"1","type":"PushEvent","created_at":"2026-08-16T10:00:00Z","repo":{"name":"octocat/playground"}},
			{"id":"2","type":"WatchEvent","created_at":"2026-08-16T09:00:00Z","repo":{"name":"octocat/other"}},
			{"id":"3","type":"IssuesEvent","created_at":"2026-08-16T08:00:00Z","repo":{"name":"octocat/playground"}}
		]`))
	}))
	defer sink.Close()

	client := testSource(nil, githubAdapter{baseURL: sink.URL, user: "octocat"})

	result, err := client.Fetch(context.Background(), "github", "pat-token", "")
	if err != nil {
		t.Fatalf("fetch github: %v", err)
	}

	if gotAuth != "Bearer pat-token" {
		t.Errorf("Authorization = %q, want the PAT as a bearer token", gotAuth)
	}
	if gotIfNoneMatch != "" {
		t.Errorf("If-None-Match = %q, want none on the first poll", gotIfNoneMatch)
	}
	if result.ETag != `W/"etag-value"` {
		t.Errorf("ETag = %q, want the weak validator returned verbatim", result.ETag)
	}

	events, ok := result.Data["events"].([]map[string]any)
	if !ok {
		t.Fatalf("events not shaped: %v", result.Data["events"])
	}
	if len(events) != 2 {
		t.Fatalf("kept %d events, want 2 (WatchEvent filtered out)", len(events))
	}
	if events[0]["type"] != "PushEvent" || events[0]["repo"] != "octocat/playground" {
		t.Errorf("first event shaped wrong: %v", events[0])
	}
	for _, e := range events {
		if e["type"] == "WatchEvent" {
			t.Errorf("WatchEvent survived the filter: %v", e)
		}
	}
}

// The prior ETag is sent as If-None-Match and a 304 is a real "no change": the
// adapter reports NotModified and echoes the ETag back so the poller keeps the
// last stored row untouched.
func TestGitHubConditionalRoundTripsETag(t *testing.T) {
	const priorETag = `W/"etag-value"`
	var gotIfNoneMatch string
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer sink.Close()

	client := testSource(nil, githubAdapter{baseURL: sink.URL, user: "octocat"})

	result, err := client.Fetch(context.Background(), "github", "pat-token", priorETag)
	if err != nil {
		t.Fatalf("fetch github 304: %v", err)
	}
	if gotIfNoneMatch != priorETag {
		t.Errorf("If-None-Match = %q, want the prior ETag sent back", gotIfNoneMatch)
	}
	if !result.NotModified {
		t.Fatal("304 did not report NotModified")
	}
	if result.ETag != priorETag {
		t.Errorf("ETag = %q, want the prior ETag preserved across a 304", result.ETag)
	}
}

// A GitHub HTTP failure is an error, not an empty feed — the unreachable-source
// invariant that keeps the health gauge from flipping.
func TestGitHubNonOKIsAnError(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer sink.Close()

	client := testSource(nil, githubAdapter{baseURL: sink.URL, user: "octocat"})

	if _, err := client.Fetch(context.Background(), "github", "pat", ""); err == nil {
		t.Fatal("github 502 returned nil error, want the failure surfaced")
	}
}

// An unknown source is a wiring bug, surfaced as an error rather than a silent
// empty result the poller would write as a state change.
func TestUnknownSourceErrors(t *testing.T) {
	client := newSourceClient("id", "user")
	if _, err := client.Fetch(context.Background(), "mastodon", "cred", ""); err == nil {
		t.Fatal("unknown source returned nil error, want a wiring error")
	}
}
