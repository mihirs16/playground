# Derived data: freshness and caching

Type: grilling
Status: open
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
