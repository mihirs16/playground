# 06 — Integration poller & timeseries read

**What to build:** persona's client-side widgets get live status without a
rebuild. One 5-minute poll tick hits both Steam and GitHub each tick on a single
loop — the same loop that reaps stale `pending` media reservations past their
`expires_at` (hence the dependency on media). The interval is config-overridable
per source with a 5-minute default. GitHub is polled with `If-None-Match`,
treating `304` as "no change". Each poll compares against the latest stored row
and inserts a new timestamped row only when the state changed; an idle/unchanged
poll inserts nothing and does not bump `updated_at`, so idle and "source briefly
unreachable" are indistinguishable by design. Every poller runs once at startup
so a row always exists before persona reads. `GET /v1/integrations/{source}`
always serves the latest row's value and its fetch timestamp (last-known-good
with an age, never an error), returns a legal empty-but-present shape when there
is no data, and carries a short TTL (`public, max-age=60, s-maxage=60,
stale-while-revalidate=300`) plus `ETag`/`If-None-Match` → `304`.

Integration sources are **code-defined adapters** (Steam, GitHub) compiled into
custodian, not runtime-registered (`24`). Their credentials (Steam key, GitHub
PAT) are **environment secrets read at startup** via `os.Getenv`, the same class
as the admin token hash and OTLP token — **not** stored in SQLite and **not**
written through the API. Rotation is replace-the-secret-and-restart. There is
**no** integration-credential write endpoint; `POST
/admin/v1/integrations/{source}/refresh` remains as an operator/debug gesture
(authed, no `broom` wrapper) that forces an immediate poll and returns the fresh
record. The Steam/GitHub clients are driven through their injected fake across
changed / unchanged / unreachable / idle / startup cases. The timeseries is kept
indefinitely; no history/timeline read endpoint in v1.

**Blocked by:** 04, 01b (needs the credential-table + endpoint removal and the
env-key config from the corrective ticket).

**Status:** done (against the injected fake; real Steam/GitHub client is ticket 11)

- [ ] One 5-minute tick polls both sources; interval config-overridable per source, 5-min default
- [ ] GitHub polled with `If-None-Match`; `304` treated as no change
- [ ] Append-on-change: new timestamped row only on state change; idle inserts nothing and does not bump `updated_at`
- [ ] Every poller runs once at startup so a row exists before first read
- [ ] Same loop reaps stale `pending` media past `expires_at`
- [ ] `GET /v1/integrations/{source}` serves latest value + fetch timestamp, empty-but-present when no data, short TTL + `304`
- [ ] Integration keys read from env (`os.Getenv`) at startup; no SQLite credential store, no credential-write endpoint
- [ ] `POST .../refresh` (operator/debug, authed) forces an immediate poll and returns the fresh record
- [ ] Poller driven through the fake across changed/unchanged/unreachable/idle/startup
