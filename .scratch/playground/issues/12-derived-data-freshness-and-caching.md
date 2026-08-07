# Derived data: freshness and caching

Type: grilling
Status: resolved
Blocked by: 02, 09

## Question

How fresh is "currently playing", and how does custodian achieve it without hammering Steam or GitHub?

- **Pull model** — does custodian poll on a schedule regardless of traffic, or lazily fetch on cache miss when persona asks? Scheduled polling gives predictable freshness and constant cost; lazy fetching costs nothing when nobody's reading but makes the first visitor pay the latency.
- **Cadence and TTL** — what interval? "Currently playing" is meaningless if it lags by an hour, but a personal site with modest traffic doesn't justify polling every thirty seconds.
- **Where the cache lives** — follows from `08`, but restate it concretely for derived data specifically, which is regenerable and therefore has weaker durability needs than authored content.
- **Staleness surfaced to the reader** — does the API return a `fetched_at` so persona can say "as of 10 minutes ago"? Strongly worth it: honest staleness beats a confidently wrong "now playing".
- **Failure behaviour** — third-party API down or rate-limited: serve last-known-good with its timestamp, serve nothing, or serve an error? For how long does last-known-good stay valid before it's actively misleading?
- **The idle case** — what does the site show when you're not playing anything and haven't pushed code recently? This is the *common* case and deserves a designed answer rather than an empty element.

## Context

Blocked on `02` (needs the researched facts: real rate limits, whether presence is even exposed, whether recently-played is the better primitive) and `09` (needs the read contract these responses fit into).

The idle case is easy to overlook and is the state a visitor most often encounters. `blank` and persona both need to know what it renders — worth coordinating with `14`.

## Answer

Most of this ticket's sub-questions were already settled upstream (`08`/`09`/`11`) and are restated here concretely; the genuinely-open decisions were cadence numbers, the edge TTL, the idle model, and the storage shape. Grilled one at a time.

**1. Poll cadence — one 5-minute tick, both sources per tick.** Quota is not the constraint (`02`: Steam 100k/day, GitHub 304s free); freshness *shape* is (Steam's recently-played is a 2-week aggregate; GitHub's feed self-lags 30s–6h). So one timer — the existing poller that also reaps stale media reservations from `09` — hits Steam and GitHub each once per tick, GitHub with `If-None-Match` (unchanged → free `304`). ~288 calls/day/source, trivially inside limits. **Interval is config (per-source overridable), 5-min default** — same static-deploy-config posture as `09`'s CORS allowlist.

**2. Pull model & cache location (restated from `09`/`08`).** Scheduled polling, **not** lazy-on-miss — predictable freshness, no first-visitor latency penalty, and the first visitor never pays an origin fetch. Derived rows live in the **same SQLite store** as everything else (`08`), riding the same S3 replication. `POST /admin/v1/integrations/{source}/refresh` (`09`) forces an immediate poll for setup/debugging.

**3. Edge/read TTL — `Cache-Control: public, max-age=60, s-maxage=60, stale-while-revalidate=300`** on `GET /v1/integrations/{source}` (`09` punted this number here). Two distinct clocks kept apart: the 5-min *poll cadence* (SQLite freshness) and the 60s *edge TTL* (CloudFront/browser). A short edge TTL well under the poll interval means the edge almost always serves the current SQLite value within a minute of it changing; `stale-while-revalidate` lets CloudFront serve slightly-stale instantly and refresh in the background, so no reader pays origin latency. Cheap: tiny JSON + `ETag`/`304` revalidation. (Rejected: matching edge TTL to the 5-min poll — up to ~10-min worst-case staleness for no cost saving at this scale.)

**4. Manual override — none in v1.** Derived data stays **purely observed** (the `03` trichotomy): the only integration write is `09`'s `/refresh`, which re-fetches — it cannot *set* a value. A pin/suppress admin override (freeze or hide a chosen value) is parked as **fog** with the "capable-of, don't-build" framing — it would blur authored-vs-derived and reopen `09`'s contract (a `pinned` flag + precedence rules), so it graduates only under a future direction.

**5. Idle case — modelled as the *absence* of a change, not a rendered state.** An idle/empty poll **does not insert and does not bump `updated_at`**; the last real value and its `ETag` stay put, so persona keeps getting `304`s and *nothing changed* from its side. Idle and "source briefly unreachable" (`11`) become indistinguishable to persona *by design* — in both cases the honest statement is "nothing new since X". All pollers run once at **startup**, so a stored row always exists by the time persona can read; an **empty array is a legal-but-rare shape** → persona renders empty-array as hide-the-widget. No `404`/omit special case. (Steam rarely goes truly idle — recently-played degrades to "last played X"; GitHub genuinely empties after 30 days quiet.)

**6. Staleness / age — custodian has zero opinion.** No max-age cutoff, no computed `stale`/`age` hint (holds `11`'s line). Custodian always carries the fetch timestamp; **persona owns the threshold** where "recently active" becomes a lie and decides whether to drop the widget or switch copy. Keeps staleness *policy* in the rendering layer — coordinates into `14`/`16`. UI copy says "recently active", never "live"/"now" (`02`).

**7. Storage shape — REVISES `08`: append-on-change timeseries, not a single upsert row.** `08` had the integration cache as one upserted JSON row per source; this ticket upgrades it to an **append-only, one-row-per-distinct-state** history so a future feature can build timelines/sparklines. Composes exactly with the idle rule: each poll compares against the latest row; **changed → insert a timestamped row; identical/idle → insert nothing.** So "idle = no update" is literally "no new row", `updated_at` = latest row's timestamp, and the series is a compact log of distinct states (not thousands of identical rows). **Read contract (`09`) is unchanged** — `GET /v1/integrations/{source}` still serves the *latest* row; any timeline/history read endpoint is **fog**.

**8. Retention — two different clocks.** The timeseries **data is kept indefinitely** (append-on-change keeps it tiny; accumulation is the point; no v1 deletion/downsampling policy). The **recovery window stays `08`'s 30 days** (WAL-shipping PITR + S3 non-current-version expiry) — that caps rewind distance, not live-data lifespan. Because history is append-only and never edited in place, the "bad write aged out of the 30-day window" risk is a near-non-issue. Adds a rounding-error to `08`'s open S3-cost caveat.

**Ripples:** refines `08` (integration storage: upsert → append-on-change timeseries; noted on that ticket). Two new fog items: the pin/override admin write, and a derived-data timeline/history read endpoint. No change to `09`'s public surface. Feeds `14`/`16` the idle + staleness rendering contract.
