package edges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// sourceClient is custodian's production SourceClient: one HTTP client shared by
// a per-source adapter, dispatched on the source string the poller passes. Each
// adapter owns its endpoint and its response-shaping; the credential and the
// prior ETag arrive per-call, resolved from the environment at startup like every
// other secret. A network failure — or any non-success, non-304 status — is
// returned as an error, never an empty result, so an unreachable source keeps its
// last-known-good row and never flips the health gauge.
type sourceClient struct {
	http     *http.Client
	adapters map[string]sourceAdapter
}

// sourceAdapter is one third party's endpoint plus the shaping of its raw
// response into persona's per-widget shape. The prior ETag rides in and the next
// one rides out on FetchResult so a conditional poll can round-trip it.
type sourceAdapter interface {
	fetch(ctx context.Context, hc *http.Client, credential, etag string) (FetchResult, error)
}

// newSourceClient builds the production client from the resolved identifiers. The
// Steam base and GitHub base default to the live hosts; tests point them at a
// local stand-in. A source with no adapter registered is a hard error at fetch,
// not a silent empty — an unknown source is a wiring bug, not a poll outcome.
func newSourceClient(steamID, githubUser string) *sourceClient {
	return &sourceClient{
		http: &http.Client{Timeout: 10 * time.Second},
		adapters: map[string]sourceAdapter{
			"steam":  steamAdapter{baseURL: "https://api.steampowered.com", steamID: steamID},
			"github": githubAdapter{baseURL: "https://api.github.com", user: githubUser},
		},
	}
}

func (c *sourceClient) Fetch(ctx context.Context, source, credential, etag string) (FetchResult, error) {
	adapter, ok := c.adapters[source]
	if !ok {
		return FetchResult{}, fmt.Errorf("no source adapter for %q", source)
	}
	return adapter.fetch(ctx, c.http, credential, etag)
}

// steamAdapter reads IPlayerService/GetRecentlyPlayedGames and collapses its
// nested two-week aggregate into a flat games list plus the summed two-week
// playtime. Steam serves no ETag, so this adapter neither sends If-None-Match nor
// returns one — every poll is a full read, which the daily-call budget easily
// absorbs (research: steam-web-api-live-status).
type steamAdapter struct {
	baseURL string
	steamID string
}

// steamMedia is the Valve-documented composition for a game's icon: the
// img_icon_url field is a filename hash under a fixed media host.
const steamMedia = "https://media.steampowered.com/steamcommunity/public/images/apps"

func (a steamAdapter) fetch(ctx context.Context, hc *http.Client, credential, _ string) (FetchResult, error) {
	endpoint := a.baseURL + "/IPlayerService/GetRecentlyPlayedGames/v0001/"
	query := url.Values{
		"key":     {credential},
		"steamid": {a.steamID},
		"format":  {"json"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return FetchResult{}, err
	}

	resp, err := hc.Do(req)
	if err != nil {
		return FetchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FetchResult{}, fmt.Errorf("steam recently-played: unexpected status %d", resp.StatusCode)
	}

	var raw struct {
		Response struct {
			Games []struct {
				AppID           int    `json:"appid"`
				Name            string `json:"name"`
				Playtime2Weeks  int    `json:"playtime_2weeks"`
				PlaytimeForever int    `json:"playtime_forever"`
				ImgIconURL      string `json:"img_icon_url"`
			} `json:"games"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return FetchResult{}, fmt.Errorf("steam recently-played: decode: %w", err)
	}

	games := make([]map[string]any, 0, len(raw.Response.Games))
	total := 0
	for _, g := range raw.Response.Games {
		total += g.Playtime2Weeks
		game := map[string]any{
			"appid":                    g.AppID,
			"name":                     g.Name,
			"playtime_2weeks_minutes":  g.Playtime2Weeks,
			"playtime_forever_minutes": g.PlaytimeForever,
		}
		if g.ImgIconURL != "" {
			game["icon_url"] = fmt.Sprintf("%s/%d/%s.jpg", steamMedia, g.AppID, g.ImgIconURL)
		}
		games = append(games, game)
	}

	return FetchResult{
		Data: map[string]any{
			"games":                         games,
			"total_playtime_2weeks_minutes": total,
		},
	}, nil
}

// githubAdapter reads a user's public events feed, filtered to the activity types
// that read as "working on" (push, PR, issue, branch/repo creation), and shaped
// into a flat feed. It is the one source polled conditionally: the prior ETag is
// sent as If-None-Match, a 304 is a real "no change", and the response ETag rides
// back out for the next poll (research: github-activity-api).
type githubAdapter struct {
	baseURL string
	user    string
}

// githubActivityTypes is the allowlist of event types that count as authored
// activity; everything else (WatchEvent, ForkEvent, …) is noise for the widget.
var githubActivityTypes = map[string]bool{
	"PushEvent":        true,
	"PullRequestEvent": true,
	"IssuesEvent":      true,
	"CreateEvent":      true,
}

func (a githubAdapter) fetch(ctx context.Context, hc *http.Client, credential, etag string) (FetchResult, error) {
	endpoint := fmt.Sprintf("%s/users/%s/events/public", a.baseURL, url.PathEscape(a.user))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return FetchResult{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return FetchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return FetchResult{NotModified: true, ETag: etag}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return FetchResult{}, fmt.Errorf("github events: unexpected status %d", resp.StatusCode)
	}

	var raw []struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		CreatedAt string `json:"created_at"`
		Repo      struct {
			Name string `json:"name"`
		} `json:"repo"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{}, fmt.Errorf("github events: read: %w", err)
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return FetchResult{}, fmt.Errorf("github events: decode: %w", err)
	}

	events := make([]map[string]any, 0, len(raw))
	for _, e := range raw {
		if !githubActivityTypes[e.Type] {
			continue
		}
		events = append(events, map[string]any{
			"id":         e.ID,
			"type":       e.Type,
			"repo":       e.Repo.Name,
			"created_at": e.CreatedAt,
		})
	}

	return FetchResult{
		// The ETag rides back out verbatim — GitHub's events feed serves a weak
		// validator (W/"…") and matches If-None-Match by weak comparison, so it
		// must see the exact token it sent. Stripping the W/ prefix would change
		// the token and suppress the 304 this conditional poll exists to get.
		ETag: resp.Header.Get("ETag"),
		Data: map[string]any{"events": events},
	}, nil
}

// ensure the concrete adapters keep satisfying the interface.
var (
	_ sourceAdapter = steamAdapter{}
	_ sourceAdapter = githubAdapter{}
)
