# Research: Steam Web API for a live-status widget

**Date**: 2026-07-25
**Question**: What can the Steam Web API give a personal-site "currently playing / recently played" widget, and what does it cost in rate limits and privacy exposure?

**Sourcing note**: `developer.valvesoftware.com` returned `403 Forbidden` on every direct fetch attempt during this research (consistent with the documented Anubis anti-bot wall). Facts attributed to that domain below were recovered via search-engine snippets of the wiki page, not a direct fetch — they are treated as primary-sourced (the wiki is Valve's own documentation) but are flagged **[fetched via search, not direct]** where that matters. `steamcommunity.com/dev`, `steamcommunity.com/dev/apiterms`, and `steamcommunity.com/dev/apikey` fetched directly without issue.

## Bottom line

- **Live "currently playing" exists** (`GetPlayerSummaries.gameid`/`gameextrainfo`) but is gated behind two independent privacy switches (profile visibility + "Game details"), and making it visible to a non-friend key **also exposes the requester's full owned-games list and per-game playtime** — there is no way to expose only the live-play field. This is the single most consequential finding for scope.
- **Recently-played (`GetRecentlyPlayedGames`) is the better primary signal for a personal site** — *judgement* — because it tolerates being read at any time (not just the exact minute of play) and, per community reports, is gated by the same profile-public requirement without necessarily requiring "Game details" public in the same way `GetOwnedGames` does. Valve's own docs don't spell out the privacy prerequisite per-endpoint, so this is partly inference (see Open/undocumented).
- **Rate limit is documented by Valve, not just community folklore**: `steamcommunity.com/dev/apiterms` states 100,000 calls/day. Polling every few minutes is trivially inside this cap (a single steamid, once every 5 minutes, is ~288 calls/day).
- **The Terms of Use do permit storing/caching Steam Data**, but only conditioned on posting a privacy policy and storing it only in disclosed countries — this is a compliance item to action, not a blocker.
- **`store.steampowered.com/api/appdetails` is confirmed undocumented by Valve** — useful for richer metadata but must be treated as best-effort/unsupported.
- **Image URLs**: the `img_icon_url`/`img_logo_url` → `media.steampowered.com` composition is Valve-documented via the Web API wiki page; the broader CDN/capsule URL patterns (`header.jpg`, `capsule_*.jpg`, akamaihd/cloudflare.steamstatic hosts) are **not found in any Valve API documentation** — they are conventional, reverse-engineered paths used by third-party tools (community-reported, not Valve-documented).

---

## 1. Live presence endpoint — `ISteamUser/GetPlayerSummaries`

Request shape, per the Steam Web API wiki (recovered via search snippet of `developer.valvesoftware.com/wiki/Steam_Web_API`) **[fetched via search, not direct]**:

```
http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=XXXXXXXXXXXXXXXXXXXXXXX&steamids=76561197960435530
```

- `key` — your Web API key.
- `steamids` — a comma-delimited list of 64-bit SteamIDs; **up to 100 SteamIDs per call**.
- Optional `format` (json default, xml, vdf) — documented on the same page (confirmed independently via the `GetRecentlyPlayedGames` example below, which shares the parameter).

Response fields relevant to "currently playing," and what each means:

| Field | Meaning | Conditional on privacy? |
|---|---|---|
| `personastate` | Current online/offline status enum (0 offline … 6 looking to trade, etc.) | Visible regardless of Game details setting, but hidden entirely if the profile itself is not public |
| `gameid` | If the user is currently in-game, the appid of that game | **Yes** — this is the "Game details" gated field |
| `gameextrainfo` | Human-readable name of the game currently being played | Same gating as `gameid` |
| `gameserverip` | IP:port of the game server the user is connected to (when applicable) | Same gating as `gameid` |
| `communityvisibilitystate` | 3 = public, other values = "not public." Community reporting notes this field only distinguishes public vs. not-public, it does not expose the granular Friends-Only-vs-Private distinction | Always returned |
| `profilestate` | 1 = user has configured a community profile, 0 = not configured | Always returned |
| `avatarfull` | Full URL to the 184×184px avatar image | Always returned if profile exists |
| `lastlogoff` | Unix timestamp of when the user was last online | Public-profile only |

Source: Steam Web API wiki page content recovered via search snippet — [developer.valvesoftware.com/wiki/Steam_Web_API](https://developer.valvesoftware.com/wiki/Steam_Web_API) **[fetched via search, not direct]**.

General privacy behavior documented on the page: "Some data associated with a Steam account may be hidden if the user has their profile visibility set to 'Friends Only' or 'Private', in which case only public data will be returned" — i.e. **fields are omitted from the response, not an HTTP error**.

## 2. Privacy prerequisites for currently-playing (non-friend key)

Two independent settings gate this, and current community reporting (not a single Valve API-doc page, but corroborated across steamcommunity discussion threads) establishes:

- **Profile visibility** must be **Public**. If Private/Friends-Only, `GetPlayerSummaries` only returns the always-public fields (`steamid`, `communityvisibilitystate`, `profilestate`, `personaname`, avatar URLs, `lastlogoff` behavior varies) — no error, just field omission. Source: Web API wiki privacy note above **[fetched via search, not direct]**.
- **"Game details"** is a *separate* profile privacy control, and as of a Steam-side change it **defaults to Friends Only for all existing accounts** (or Private if the whole profile is Private) — it does not automatically follow "profile public." This is reported in Steam Community help threads, not stated on a Valve API doc page: [steamcommunity.com/discussions/forum/7/1729827777339922602](https://steamcommunity.com/discussions/forum/7/1729827777339922602/) — **not documented by Valve on the API reference itself; community-reported**.
- Practical consequence reported by users and third-party API wrapper maintainers: `GetOwnedGames` (and by the same gating mechanism, the in-game fields of `GetPlayerSummaries`) return an **empty/omitted result when Game details is not public**, not an error — [steamcommunity.com/discussions/forum/7/1729827777339922602](https://steamcommunity.com/discussions/forum/7/1729827777339922602/), **community-reported**.
- Important exception: **the constraint above is for a key not linked to the target account.** If the WebAPI key you're using is the same account you're querying (the personal-site use case), the owner's own key against their own SteamID bypasses these prerequisites entirely — same source.

### Side effect of making "Game details" public

This matters because the user explicitly wants to know the blast radius. Making "Game details" Public does **not** selectively expose only live-play — it is the single switch that also unlocks, for any caller (not just friends):

- **Full owned-games list** via `GetOwnedGames` — every game the account owns becomes enumerable, not just the one currently played.
- **Per-game playtime** (`playtime_forever`, and 2-week playtime via `GetRecentlyPlayedGames`) for every owned game, i.e. hours-played history becomes public.

This is documented behavior of `GetOwnedGames` requiring public visibility (Valve wiki, **[fetched via search, not direct]**), corroborated by community reporting on the Game details toggle specifically gating it: [steamgifts.com/discussion/RmZt0](https://www.steamgifts.com/discussion/RmZt0/steam-developer-web-api-getownedgames-and-privacy-settings) and [steamcommunity.com/discussions/forum/7/1729827777339922602](https://steamcommunity.com/discussions/forum/7/1729827777339922602/) — **community-reported, not a single Valve statement**.

Wishlist and inventory are **separate systems** (different endpoints, e.g. `IEconItems`/inventory service, and the wishlist is not exposed by any documented Steam Web API method found in this research) — no evidence found, in either direction, that toggling Game details affects wishlist or inventory visibility. Treat as **out of scope / not established** rather than assuming exposure.

## 3. Rate limits and Terms of Use

- **100,000 calls/day** is stated directly in Valve's own Terms of Use page: "You are limited to one hundred thousand (100,000) calls to the Steam Web API per day. Valve may approve higher daily call limits if you adhere to these API Terms of Use." — [steamcommunity.com/dev/apiterms](https://steamcommunity.com/dev/apiterms). This is a primary Valve statement, directly fetched, not a community-inferred number.
- **Caching/storing is permitted, conditionally.** The TOU requires that you "post a privacy policy regarding the use of nonpublic end user data (including such Steam Data)" and that you store Steam Data "in a country (or countries) identified in your privacy policy," and that you "will only retrieve Steam Data about a Steam end user as requested by the end user." — [steamcommunity.com/dev/apiterms](https://steamcommunity.com/dev/apiterms). There is no blanket "no caching" clause; the obligation is disclosure, not prohibition. For a personal site querying only the owner's own account, "requested by the end user" is trivially satisfied by the owner being the operator.
- **Attribution/display restriction**: you may not present the data such that your application "appears (a) endorsed or affiliated with Valve or Steam, or (b) to be available from a third party," and Valve-branding requirements apply, with a specific prohibition on `nofollow` on links to Valve. — [steamcommunity.com/dev/apiterms](https://steamcommunity.com/dev/apiterms).
- **Polling frequency**: the TOU does not name a per-minute or per-second cap — the only documented ceiling is the 100,000/day figure. Polling a single account every few minutes (roughly 200–300 calls/day) is comfortably inside the documented limit. *Judgement*: nothing in the TOU forbids this cadence; the daily cap is the only quantitative constraint Valve states.

## 4. Live presence vs. recently-played

`IPlayerService/GetRecentlyPlayedGames`:

- Example call: `http://api.steampowered.com/IPlayerService/GetRecentlyPlayedGames/v0001/?key=XXXXXXXXXXXXXXXXX&steamid=76561197960434622&format=json` — [developer.valvesoftware.com/wiki/Steam_Web_API](https://developer.valvesoftware.com/wiki/Steam_Web_API) **[fetched via search, not direct]**.
- Params: `steamid` (required), `count` (optional, limits number of games returned), `format`.
- Response `games[]` fields: `appid`, `name`, `playtime_2weeks` (minutes played in the last 2 weeks), `playtime_forever` (minutes since Steam began tracking, early 2009), `img_icon_url`, `img_logo_url`.

`IPlayerService/GetOwnedGames`:

- Example call: `http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=XXXXXXXXXXXXXXXXX&steamid=76561197960434622&format=json` — same source.
- Params: `steamid` (required), `include_appinfo` (bool, includes name/logo — default is appids only), `include_played_free_games` (bool, includes free games like TF2 that would otherwise be excluded since "everyone owns them"), and an appid filter that per the docs "cannot be passed as a URL parameter" and instead requires the JSON-input calling convention for service interfaces.

Assessment (*judgement*): for a personal site that is read asynchronously (visitors don't land at the exact moment of play), **`GetRecentlyPlayedGames` is the stronger primary signal**. It degrades gracefully — "played X two weeks ago" is a reasonable widget state even when nothing is live — whereas `GetPlayerSummaries`'s `gameid`/`gameextrainfo` is only non-empty during the narrow window the account is actually in a running game, and empty otherwise with no distinction from "privacy blocked" vs. "not currently playing." `GetRecentlyPlayedGames` also, per the community reporting in section 2, is understood to inherit the same public-profile prerequisite as `GetOwnedGames` rather than exposing a separate slice — **not explicitly confirmed by Valve docs for this specific endpoint**, flagged as inference.

Both endpoints, for a non-owner key, are read as requiring the same "profile public + Game details public" pair described in section 2. For the personal-site case (owner's own key against their own account) neither restriction applies.

## 5. Artwork

- **Documented by Valve**: `img_icon_url` / `img_logo_url` are filenames (hashes), composed into a URL as `http://media.steampowered.com/steamcommunity/public/images/apps/{appid}/{hash}.jpg` per the Web API wiki — [developer.valvesoftware.com/wiki/Steam_Web_API](https://developer.valvesoftware.com/wiki/Steam_Web_API) **[fetched via search, not direct]**.
- **Not found in any Valve API documentation**: `header.jpg`, `capsule_231x87.jpg`, `capsule_616x353.jpg`, `library_600x900.jpg` path conventions under `steamcdn-a.akamaihd.net` or `shared.cloudflare.steamstatic.com`/`cdn.cloudflare.steamstatic.com`. These are widely used by third-party tooling (SteamDB, community wrappers) but no Valve API reference describing them was found — **not documented by Valve; community-reported convention**.
- **Store front asset specs** (dimensions, content rules for what may appear on capsule art) are documented, but on the *Steamworks partner* side for developers submitting art, not as a public consumption API: [partner.steamgames.com/doc/store/assets/rules](https://partner.steamgames.com/doc/store/assets/rules), [partner.steamgames.com/doc/store/assets/standard](https://partner.steamgames.com/doc/store/assets/standard). These describe what Valve requires *from game developers*, not terms for third parties displaying the resulting images.
- **Hotlinking / display permission**: no explicit statement found, in the Web API Terms of Use or the Steamworks branding docs, permitting or forbidding hotlinking Steam CDN artwork on a third-party site. The apiterms document's only relevant clause is the non-affiliation/non-endorsement restriction described in section 3, which is about not implying affiliation — not specifically about image hotlinking. **Not documented by Valve either way; treat as unaddressed**, and mitigate by proxying/caching images server-side (also aligns with the "your Application... appears... available from a third party" clause, which arguably favors self-hosting a copy over hotlinking).
- The Steam Branding Guidelines PDF governs the Steam *logo* specifically ("must stand alone," no combination with other marks) — [partner.steamgames.com/public/marketing/Steam_Guidelines_01042017.pdf](https://partner.steamgames.com/public/marketing/Steam_Guidelines_01042017.pdf) — this is about the Steam logo/brand mark, not game cover art, and does not resolve the hotlinking question above.

## 6. Key acquisition

- Obtained at [steamcommunity.com/dev/apikey](https://steamcommunity.com/dev/apikey/).
- **Account must not be "Limited."** Community reporting: Limited status is lifted once the account has spent at least $5 on Steam (game purchase, wallet funds, etc.) — [steamcommunity.com discussions](https://steamcommunity.com/discussions/forum/1/135511655649987528) — **community-reported figure, not stated in Valve's own apikey page copy fetched during this research**; the apikey page itself did not state a numeric threshold in what was retrievable.
- **Domain Name field**: intended to record where the key will be used; commonly reported as not strictly enforced, and using `localhost` is a widely accepted value for personal/dev projects — community-reported, not enforced per Valve's own copy as far as could be established.
- **Steam Guard / mobile authenticator confirmation** is required to create a key (an added security step) — corroborated by direct fetch of `steamcommunity.com/dev/apikey`, which returned account-security-flow content (Steam Guard, mobile authenticator) consistent with this gating, though the fetch tool did not surface the literal apikey-registration copy verbatim.
- **A single key is adequate for a server-side poller** — nothing in the documented Terms of Use or the registration flow implies one key per steamid or per client; the key is tied to the *registering* account and rate-limited at 100,000 calls/day in aggregate, which comfortably covers polling one (or a handful of) accounts every few minutes.

## 7. Store API (`store.steampowered.com/api/appdetails`)

**Confirmed undocumented by Valve.** Community sourcing (which is itself explicit about Valve's silence): "The storefront API is completely undocumented by Valve and was developed for their own use (such as Big Picture mode), with zero official word from Valve on usage limits or 3rd party usage" — this characterization appears in secondary community write-ups (e.g. referencing `steamapi.xpaw.me` and the `Revadike/InternalSteamWebAPI` community documentation project), not on any Valve property. **Not documented by Valve; community-reported / reverse-engineered.**

Implication for the widget: usable for richer metadata (categories, genres, release date, F2P flag) as a *best-effort enhancement*, but must not be a hard dependency — no Valve-documented stability or rate-limit guarantee exists, unlike the official Web API endpoints covered above.

---

## Open / undocumented

Findings that could not be established from primary sources within this research, or where Valve is silent:

- **Exact per-endpoint privacy prerequisite is not spelled out by Valve.** The Web API wiki's privacy language is general ("some data... may be hidden"); the specific claim that `GetRecentlyPlayedGames` and the in-game fields of `GetPlayerSummaries` require *both* profile-public *and* Game-details-public (vs. just one) rests on community reporting and inference by analogy to `GetOwnedGames`, not a direct Valve statement per endpoint.
- **Whether "Game details" public also affects wishlist or inventory visibility** — no evidence found either way; not established.
- **Whether hotlinking Steam CDN artwork is explicitly permitted or forbidden** — neither the Web API Terms of Use nor the Steamworks branding docs address this directly.
- **The `header.jpg`/`capsule_*.jpg`/`library_600x900.jpg` CDN path conventions** — no Valve API documentation found describing these; entirely community/reverse-engineered.
- **Exact numeric threshold to lift "Limited" account status** (commonly cited as $5 spent) — this figure comes from Steam Community discussion threads, not a Valve policy page fetched during this research.
- **`developer.valvesoftware.com` wiki content in this report was recovered via search-engine snippets, not direct fetch**, because every direct `WebFetch` attempt against that domain returned HTTP 403 (consistent with the documented Anubis anti-bot wall). The underlying page is still Valve's own wiki — treated as primary — but the specific request/response field descriptions above should be spot-checked against the live page if and when it becomes fetchable, since snippet-based recovery cannot guarantee completeness (e.g. exact enum values for `personastate` were not fully enumerated in what was retrieved).
- **Whether the Domain Name field on the apikey registration form is enforced in any way** (e.g. CORS, referer checks) — not established from the direct fetch, which did not surface the registration form's own copy verbatim.
