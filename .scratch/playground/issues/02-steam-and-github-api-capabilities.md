# Steam and GitHub API capabilities

Type: research
Status: resolved

## Question

What can the Steam and GitHub APIs actually give us for the **derived** content bucket, and what do they cost in rate limits and privacy constraints?

Specifically:

**Steam — "currently playing"**
- Which endpoint exposes it? (`ISteamUser/GetPlayerSummaries` and its `gameextrainfo` / `gameid` fields are the likely candidates.)
- What privacy settings must the profile have for presence to be visible at all? Does "Game details" need to be public, and does that leak anything unwanted?
- Rate limits, key acquisition, and terms — is polling every few minutes acceptable use?
- Is recently-played (`IPlayerService/GetRecentlyPlayedGames`) a better fit than live presence, given a personal site is rarely being read at the exact moment you're in a game?
- Is there any richer artwork available (capsule / header images) and what are the URL patterns and licensing implications?

**GitHub — "currently working projects"**
- Which is the right source: the public events feed, recent pushes, contribution activity, or repo `pushed_at`? What does each actually reflect, and how quickly does each update?
- Does it require auth for reasonable rate limits, and what scope is the minimum for public activity only?
- Can private-repo activity be surfaced without leaking repo names, if that's ever wanted?
- Rate limits for unauthenticated vs token-authenticated requests.

## Context

AFK — resolve with a `/research` subagent against primary sources (Steam Web API docs, GitHub REST/GraphQL docs). No decisions here; this ticket produces the facts that `12` (freshness and caching) needs before it can decide a polling strategy.

The live-status feature is one of the two things the site is being rebuilt for, so if Steam turns out to be materially more limited than assumed — presence hidden behind privacy settings, or unusable rate limits — that's a finding worth surfacing early rather than discovering during implementation.

## Blocks

`12` derived-data freshness and caching

## Answer

Full findings: [Steam Web API live status](../../../docs/research/steam-web-api-live-status.md), [GitHub activity API](../../../docs/research/github-activity-api.md).

**Steam.**
- `ISteamUser/GetPlayerSummaries` (`api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/`, up to 100 SteamIDs/call) does expose currently-playing (`gameid`, `gameextrainfo`, `gameserverip`), but only when queried by a key *not* linked to the account does the profile's privacy apply — a self-owned key querying the owner's own account bypasses the gating entirely, so this is moot for custodian's use case.
- For any key that isn't self-owned, "Game details" now **defaults to Friends-Only**, and there is no way to expose just live-play — flipping it Public also enumerates the entire owned-games library and per-game playtime via `GetOwnedGames`. Not relevant to a self-key poller, but worth recording as a constraint if the key or query pattern ever changes.
- **Judgement**: `IPlayerService/GetRecentlyPlayedGames` (`appid`, `name`, `playtime_2weeks`, `playtime_forever`, `img_icon_url`) is the better primary signal for a site rarely read at the exact moment its owner is playing — it degrades to "played X two weeks ago" instead of going empty the instant the game closes.
- Rate limit: **100,000 calls/day**, confirmed directly from Valve's Terms of Use (`steamcommunity.com/dev/apiterms`) — a real primary-source citation, not folklore. Polling every few minutes (~200–300 calls/day) is trivially inside this. Caching/storing responses is permitted, conditioned on disclosing storage and a privacy policy.
- Artwork: `img_icon_url`/`img_logo_url` via `media.steampowered.com` is Valve-documented; `header.jpg`/`capsule_*.jpg`/CDN-host URL conventions are **not** documented by Valve — community convention only, hotlinking permission unaddressed either way.
- `store.steampowered.com/api/appdetails` is confirmed undocumented — usable only as best-effort enhancement, never a dependency.
- Caveat: `developer.valvesoftware.com` returned 403 (Anubis anti-bot wall) on every direct fetch; wiki-sourced claims were recovered via search snippets, flagged inline in the research doc.

**GitHub.**
- **Judgement**: `GET /users/{username}/events/public`, polled every 1–2 min with `If-None-Match` ETag conditional requests, authenticated with a fine-grained PAT granted **zero permissions** — this is the only candidate reflecting actual activity rather than a coarse timestamp or annual aggregate.
- **Most important finding**: GitHub explicitly disclaims real-time behavior on this endpoint — event latency can run 30s to 6h depending on time of day. Custodian's UI should read "recently active," never "right now." The endpoint caps at 300 events and a **30-day** lookback (not the commonly assumed 90 days — corrected during research).
- Rate limits: unauthenticated 60 req/hr/IP; authenticated (PAT/OAuth/App) 5,000 req/hr. GraphQL has a separate 5,000-points/hr budget. **304 Not Modified responses from conditional requests don't count against the primary rate limit when authenticated** — frequent polling is effectively free at hobby scale.
- A fine-grained PAT with *no permissions selected* still gets full read-only access to all public repos/activity at the 5,000/hr tier — no scope needs requesting for a public-only feed.
- Private-repo activity can be surfaced as counts-only via GraphQL `contributionsCollection.restrictedContributionsCount`, gated by the account-level "Include private contributions on my profile" setting rather than token scope — the poller's token can safely stay public-only regardless.
- `pushed_at` semantics are undocumented by GitHub (typed, not described) — flagged as community-reported only.
- No ToS restriction found on caching/displaying one's own public activity; GitHub's Acceptable Use Policy explicitly states API usage isn't "scraping." (Full ToS Section H not fetched — noted as an open item in the research doc, not expected to change the recommendation.)

**Net effect on `12` (derived-data freshness and caching)**: both APIs support a poll-and-cache model well within their rate limits at hobby scale; the constraint isn't quota, it's the *shape* of freshness — Steam's live-presence field can go stale the instant a session ends (recently-played is the fallback), and GitHub's events feed carries its own documented multi-hour lag ceiling. Ticket `12` should design its cache TTL and its UI copy ("recently active" not "live") around those, not around rate limits.
