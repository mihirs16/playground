# Spec: custodian

Status: ready-for-agent

The API that keeps the playground. `custodian` owns all content, serves it to
`persona` (at build time for baked content, at runtime for derived widgets),
accepts writes from `broom`, and is the only component that holds third-party
credentials.

This spec synthesises the resolved decision tickets `07` (language & runtime),
`08` (storage model), `09` (API contract), `10` (auth model), `11`
(observability), and `12` (derived-data freshness), grounded in the `03` domain
model, ADR-0001 (hosting posture), and the downstream revisions from `19` (deed
provisioning), `23`/`15` (persona blog delivery), and `16` (blog SEO). Where a
later ticket revised an earlier one, the revised position is stated here as the
single source of truth.

## Problem Statement

The author writes and maintains a personal site: long-form blog posts, a
profile (about / experience / skills / resume link / curated projects), and a
pair of live-status widgets (Steam currently-playing, GitHub activity). The
deprecated site made this painful in two specific ways:

- **Editing meant rebuilding.** Content lived in code/config consumed by
  `getStaticProps` with no revalidation, so every content edit required a full
  site rebuild and redeploy. Fixing a typo was a deploy.
- **A secret was shipped to the browser.** The Notion integration token was set
  in `NEXT_PUBLIC_NOTION_KEY`, which inlined it into the client bundle and
  handed it to every visitor.

The author wants a single place that owns all content, that can be edited from
the terminal without touching the site's build, and that keeps every
third-party credential server-side — while running cheaply and staying up,
because the site's primary content depends on it.

## Solution

`custodian` is a single always-on Go service on one EC2 `t4g.micro`, fronted by
nginx and CloudFront (ADR-0001). It exposes a REST/JSON API over two surfaces in
one binary:

- A **public read surface** (`/v1/*`) — unauthenticated, GET-only, cacheable —
  that `persona` reads to bake blog prose, the blog index, and profile content
  at build time, and that `persona`'s client-side widgets read at runtime for
  live status.
- An **admin surface** (`/admin/v1/*`) — authenticated, full CRUD,
  uncacheable — that `broom` uses to author and manage content.

Content lives in an in-process SQLite database that is the sole source of truth,
continuously WAL-shipped to S3 for point-in-time recovery; uploaded media bytes
live in S3 and are served from a CDN URL that never passes through custodian.
Third-party credentials never leave the box: custodian polls Steam and GitHub
server-side on a timer and caches the results, so `persona` never calls a
third-party API and never receives a key.

Editing is now a terminal gesture against a live API, not a rebuild — the
rebuild-to-edit problem is solved for the act of *writing*. (Persona later
re-introduces a manual rebuild to *publish* baked blog content; that is
persona's editorial-batching choice, external to custodian, which always serves
current content the instant it is written.)

## User Stories

### Reading — public surface (`persona`, at build time and runtime)

1. As `persona`'s build, I want to fetch a paginated index of listed logs, so that I can generate the blog index page.
2. As `persona`'s build, I want each log summary in the index to omit the body, so that the index payload stays small.
3. As `persona`'s build, I want the index to accept `limit`/`offset` and an optional `tag` filter, so that I can page and filter without a contract change as post count grows.
4. As `persona`'s build, I want the index to return a `total` count alongside the items, so that I can render pagination controls.
5. As `persona`'s build, I want to fetch a single log by its slug and receive its full body, so that I can bake the post page.
6. As `persona`'s build, I want to fetch an `unlisted` log by slug and still receive it, so that a draft is previewable at its real URL before it is listed.
7. As `persona`'s build, I want to fetch a profile record by key (`about`, `experience`, `skills`, `resume-link`, `curated-projects`), so that I can bake profile content into the site.
8. As `persona`'s build, I want a failed custodian fetch to fail the build loudly, so that I never silently ship a site with missing baked content.
9. As a `persona` client-side widget, I want to fetch the latest integration record for a source (`steam`, `github`) at runtime, so that I can render live status without a rebuild.
10. As a `persona` client-side widget, I want each integration record to carry a fetch timestamp, so that I can decide myself when "recently active" has become a lie.
11. As a `persona` client-side widget, I want an integration with no data to be a legal, empty-but-present shape, so that I can hide the widget rather than error.
12. As any public reader, I want every read response to carry an `ETag` and `Cache-Control`, so that CloudFront and my browser can cache and revalidate.
13. As any public reader, I want custodian to honour `If-None-Match` and return `304 Not Modified` with an empty body when nothing changed, so that repeat reads are cheap.
14. As `persona`, I want logs and profile to carry long, revalidate-friendly cache headers and integrations to carry a short TTL, so that baked content caches hard while live status stays fresh.
15. As a browser making a cross-origin read, I want the public surface to allow my origin explicitly (allowlist, not wildcard) and expose `ETag`, so that conditional requests work from the browser.
16. As `persona`, I want to fetch media bytes directly from the CDN URL and never from the API, so that image delivery never touches custodian's process.

### Writing — admin surface (`broom`)

17. As `broom`, I want to create a log with an author-chosen slug via `POST`, so that a new post exists immediately as an `unlisted` draft.
18. As `broom`, I want a slug collision on create to return a `409` with a stable `slug_conflict` code, so that I can offer the author a different slug.
19. As `broom`, I want to partially update a log via `PATCH` where `state`, `slug`, `body`, and metadata are ordinary fields, so that I never resend the whole record to change one thing.
20. As `broom`, I want publishing to be `PATCH {state:"listed"}` and unpublishing to be `PATCH {state:"unlisted"}`, so that there are no bespoke publish/unpublish endpoints.
21. As `broom`, I want to rename a log's slug via `PATCH` while it is `unlisted`, and have custodian perform the move and echo the new slug back, so that I can track the post by its new identity.
22. As `broom`, I want a slug-rename attempt on a `listed` log to be rejected with a stable `slug_frozen_while_listed` code, so that published links never break.
23. As `broom`, I want to delete a log via `DELETE`, so that I can remove a post entirely.
24. As `broom`, I want to fetch any log (including unlisted drafts) and list logs of any state on the admin surface, so that I can manage drafts the public index does not show.
25. As `broom`, I want to reserve a media record with a key via `POST /admin/v1/media` and receive a presigned S3 upload URL plus the public CDN URL, so that I can upload bytes straight to S3 without any AWS credentials.
26. As `broom`, I want media-key uniqueness enforced at reserve time, so that a duplicate key errors (`media_key_taken`) rather than silently overwriting.
27. As `broom`, I want to omit the key and have custodian generate a random kebab-case key, so that I can upload without naming everything.
28. As `broom`, I want to confirm an upload via `POST /admin/v1/media/{key}/confirm` and have custodian `HEAD` S3 before flipping the record to `available`, so that every `available` media record is guaranteed to have real bytes behind it.
29. As `broom`, I want to list and search media, so that I can find and reuse an existing asset.
30. As `broom`, I want to delete a media record, so that I can remove an unused asset (the pre-delete reference scan is `broom`'s own courtesy, since custodian does not parse bodies for URLs).
31. As `broom`, I want to upsert a profile record by key via `PUT` with an opaque JSON body that custodian does not validate, so that I control the profile shape by convention with `persona`.
32. As `broom`, I want to force an immediate poll of an integration via `POST /admin/v1/integrations/{source}/refresh` and get the fresh record back, so that I can verify a newly rotated Steam key or fixed GitHub PAT.
33. As `broom`, I want to write a third-party integration credential through the authed admin API and have it take effect on the next poll with no restart, so that key rotation is a terminal gesture, not a redeploy.
34. As `broom`, I want every error as RFC 9457 `application/problem+json` with a stable `code` and, for validation failures, a field-errors array, so that I can print `detail`, branch on `code`, and list field problems.

### Derived data — polling and caching

35. As custodian, I want one 5-minute poll tick that hits both Steam and GitHub each tick, so that freshness is predictable and independent of traffic.
36. As custodian, I want the poll interval to be config-overridable per source with a 5-minute default, so that cadence is tunable without a code change.
37. As custodian, I want to poll GitHub with `If-None-Match` and treat a `304` as "no change", so that I stay well inside GitHub's rate limits for free.
38. As custodian, I want to compare each poll against the latest stored row and insert a new timestamped row only when the state changed, so that the store is an append-on-change timeseries, not thousands of identical rows.
39. As custodian, I want an idle/unchanged poll to insert nothing and not bump `updated_at`, so that idle and "source briefly unreachable" are indistinguishable to `persona` by design — the honest statement in both cases is "nothing new since X".
40. As custodian, I want to run every poller once at startup, so that a stored row always exists by the time `persona` can read.
41. As custodian, I want the read surface for an integration to always serve the latest row's value and its fetch timestamp, so that `persona` gets last-known-good with an age, never an error, when a source is down.
42. As custodian, I want to keep the integration timeseries indefinitely (it stays tiny), while the recovery window stays 30 days, so that future timelines are buildable without a v1 history endpoint.
43. As custodian, I want to reap stale `pending` media reservations past their `expires_at` on the same poll loop, so that no new background machinery is needed.

### Auth and abuse

44. As custodian, I want auth to guard only `/admin/*`, so that the public read surface stays credential-free and CORS-simple.
45. As custodian, I want to accept a single long-lived bearer token on `Authorization: Bearer` and store only its hash, so that a DB or S3 leak yields nothing usable.
46. As custodian, I want the credential to ride a header and never a cookie, so that the public surface's CORS allowlist never has to reason about credentialed requests.
47. As the author, I want token rotation to be replace-the-secret-and-restart with implicit revocation (the old hash stops matching), so that there is no token registry and no self-credential endpoints to attack.
48. As custodian, I want to read the admin token hash and the OTLP export token from my process environment at startup and treat their source as opaque, so that `deed` can swap the concrete secret store without touching my code.
49. As custodian, I want to log every `/admin/*` access (timestamp, real client IP, path, result), so that any admin call the author did not make is a visible leak signal.
50. As the operator, I want nginx `limit_req` per-IP and per-location (tighter on `/admin/*`, looser on `/v1/*`) using the real client IP from CloudFront's `X-Forwarded-For`, so that origin hammering is throttled without putting rate-limiting logic in Go.
51. As the account owner, I want an AWS WAF rate-based rule on the CloudFront distribution plus a Budgets alarm, so that an edge-cache wallet-DoS is dropped at the edge during the attack (Budgets alone cannot throttle traffic).

### Observability

52. As custodian, I want to self-assess health on a timer — `SELECT 1` on SQLite, an S3 `HeadBucket`, and disk/memory headroom — and emit a single `health` gauge (`1` healthy / `0` degraded), so that health is one clear signal.
53. As custodian, I want third-party API reachability excluded from health entirely, so that Steam being down never turns custodian red.
54. As custodian, I want to embed the OTel Go SDK and export metrics, traces, and logs over OTLP/HTTP directly to a configured endpoint, so that there is no agent sidecar on the 1 GB box and the backend is a swappable config URL.
55. As the operator, I want the `health` gauge to double as the heartbeat, with one alert rule firing on `health == 0` OR no-data-for-a-window, so that degraded and down are the same rule and it reaches me by mobile push.
56. As the operator, I want a trivial debug-only `/healthz` endpoint for manual `curl`, so that health is inspectable without it being a load-bearing public contract.

### Recovery

57. As the author, I want SQLite continuously WAL-shipped to S3 with per-timestamp point-in-time restore over a 30-day window, so that "I ran a bad write an hour ago" is recoverable, not just "the instance died".
58. As the author, I want S3 bucket versioning on both the DB-replica bucket and the media bucket with 30-day non-current-version expiry, so that a bad object delete or overwrite is undoable.

## Implementation Decisions

### Language, runtime, and deploy shape (`07`, ADR-0001)

- **Go**, compiled to a single static binary. Chosen as the boring,
  debuggable-at-11pm choice for the one component that must stay up, and because
  it is career-aligned learning for the author (maintenance risk and learning
  motivation point the same way).
- **SQLite driver: pure-Go `modernc.org/sqlite`**, so `CGO_ENABLED=0` and the
  ARM64 binary cross-compiles from the author's laptop with no C toolchain.
- **HTTP stack: chi.** A `chi.Router` *is* an `http.Handler`; no framework
  lock-in, composable middleware for auth/logging/recovery, sub-routers for the
  two surfaces.
- **AWS SDK for Go v2** for the IMDS instance-profile credential chain — no
  long-lived AWS credentials on disk (ADR-0001).
- Runs comfortably in 1 GB on ARM64; no day-one resize.
- Deploy shape per `19`: the binary runs as a container under `docker compose`
  on one EC2 box (this supersedes ADR-0001's original "systemd unit" wording),
  behind nginx, with Litestream as a sidecar. custodian's code is agnostic to
  this; it reads only `os.Getenv` for its secrets.

### Storage model (`08`, revised by `12`)

- **SQLite is the sole source of truth.** The `log` body (GFM-core text) is a
  column. No git backing, no dual source of truth.
- **`log` table**: title, subtitle, slug, cover image, reading time, tags,
  created_at, updated_at, body, state, and an optional plain-prose
  **`description`** field (graduated from fog by `16` — see below). Edits
  overwrite in place; `created_at`/`updated_at` track timing.
- **Version history is deferred** (additive fog). A `log_revision` table is
  purely additive, so shipping without it costs nothing structurally.
- **Media key is a single kebab-case string** — author-provided or
  random-kebab. Uniqueness is enforced (duplicate → error, never a silent
  overwrite). The public URL is extension-free; content-type is served from
  record metadata via CDN headers. Content-addressing was rejected.
- **`profile` table**: one row per key (`about`, `experience`, `skills`,
  `resume-link`, `curated-projects`), each an `id` plus an opaque JSON `body`.
  custodian does not validate the body shape.
- **`integration` storage is an append-on-change timeseries** (revised from
  `08`'s single upsert row by `12`): one row per distinct polled state, each
  timestamped. Each poll compares against the latest row — changed → insert;
  identical/idle → insert nothing. The read surface serves the latest row. Data
  kept indefinitely; a history/timeline read endpoint is fog.
- **Media bytes live in S3**; the DB replica and media both ride S3 with
  versioning. **Recovery**: Litestream-style WAL shipping for per-timestamp
  PITR + S3 versioning, 30-day retention on both buckets. *Open caveat carried:
  confirm the 30-day-retention S3 cost against ADR-0001's ~£10/mo envelope when
  buckets are configured.*
- Writes are **fully synchronous** (SQLite in-process) — the API need not model
  async or commit-shaped writes.

### API contract (`09`)

- **Style: REST/JSON.** GraphQL and RPC both rejected (they fight HTTP edge
  caching and same-origin GETs).
- **Topology: two surfaces, one Go binary, split by audience not verb.** Public
  `/v1/*` (unauth, GET-only, cacheable, CORS allowlist) and admin `/admin/v1/*`
  (authenticated, full CRUD, `no-store`, no CORS). The split is public-vs-admin
  because `broom` reads too (unlisted drafts, media search). Path-prefix split
  makes CloudFront cache-all-vs-cache-none behaviours trivial and means auth
  only ever guards `/admin/*`.
- **Contract format: OpenAPI 3.1, spec-first.** A hand-authored `.yaml` is the
  source of truth; Go server interfaces (oapi-codegen) and consumer clients
  (openapi-typescript / the Go client for `broom`) generate from it.
  Per `13`, the `.yaml` lives under custodian and `just gen` fans a vendored,
  committed, CI-drift-checked client into each consumer. **Authoring the
  OpenAPI `.yaml` is part of this spec's implementation work** — `09` recorded
  request/response sketches but deliberately built no artifact.
- **Endpoint surface:**
  - Public: `GET /v1/logs`, `GET /v1/logs/{slug}`,
    `GET /v1/integrations/{source}`, `GET /v1/profile/{key}`.
  - Admin: `GET|POST /admin/v1/logs`, `PATCH|DELETE /admin/v1/logs/{slug}`;
    `GET|POST /admin/v1/media`, `GET|DELETE /admin/v1/media/{key}`,
    `POST /admin/v1/media/{key}/confirm`; `PUT /admin/v1/profile/{key}`;
    `POST /admin/v1/integrations/{source}/refresh`; plus the admin
    integration-credential write (see auth, below).
- **Read shapes:** index → `{ total, items: [summary] }`, summaries omit `body`,
  listed-only, offset/limit pagination with optional `tag`. Detail → full log
  including `body`, any state. `persona` never fetches media from the API.
- **Caching mechanism:** every read carries `ETag` + `Cache-Control`; custodian
  honours `If-None-Match` → `304`. ETag inputs: `updated_at` for logs (also a
  free `Last-Modified`), fetch timestamp for integrations. Per-type
  `Cache-Control`: logs/profile long + revalidate; integrations short (numbers
  from `12`, below).
- **CORS:** explicit origin allowlist (not wildcard), exposes `ETag`, allows
  `If-None-Match`; allowlist lives in static deploy config (env/file), not a
  `broom`-managed resource. Admin surface: no CORS.
- **Media upload = presigned S3 `PUT`, custodian off the byte path.**
  `POST /admin/v1/media` reserves a `pending` record (key uniqueness enforced
  now) → returns `{ upload_url, url, expires_at }` → `broom` `PUT`s bytes to S3
  → `POST .../confirm` → custodian `HEAD`s S3 → flips to `available`. Invariant:
  every `available` record has real bytes. Stale `pending` records are reaped by
  the poller.
- **Log identity: slug is sole identity.** Mutable while `unlisted`, frozen once
  `listed`. Rename is a server-performed move via `PATCH` with a new slug.
- **Log write verbs: `POST` + `PATCH` + `DELETE`.** No full-replace `PUT`, no
  bespoke `:list`/`:unlist` actions.
- **Profile: `PUT`-upsert opaque JSON, no validation.** Integrations:
  poller-fetched, the only refresh write is `POST .../refresh`.
- **Errors: RFC 9457 `application/problem+json`** with a stable `code`
  extension (`slug_conflict`, `media_key_taken`, `log_not_found`,
  `slug_frozen_while_listed`, …) and a field-errors array for validation.
- **Versioning: URL path `/v1`, additive-only within a major**, OpenAPI
  `info.version` semver; a breaking change is `/v2` beside `/v1` during cutover.
- **Body is stored verbatim** — custodian does no GFM validation or rendering;
  rendering is `persona`/`blank`'s job.

### The `description` field (`16`, revises `03`/`08`/`09`)

- A dedicated **optional plain-prose `description`** field graduates onto the
  `log` record, used by `persona` for SEO/OG `description` meta. `subtitle`
  stays editorial (a preface/quote) and must **not** be conscripted as an SEO
  summary. Missing description → `persona` omits the meta tag.

### Auth model (`10`, revised by `19`)

- **Single hashed bearer token** on `Authorization: Bearer`; custodian stores
  only the hash. No OAuth, mTLS, or login server (multi-user ceremony a single
  writer does not need); the upgrade path is fog, not foreclosed.
- **Two log states only** — `listed` (indexed + reachable) and `unlisted`
  (reachable-by-slug, excluded from index). Every fetched log is public; draft
  privacy rests on slug unguessability. **No authenticated-preview third
  state** — inventing one would force `persona` to send a credential from the
  browser, exactly the `NEXT_PUBLIC_NOTION_KEY` trap.
- **Secret model splits in two (revised by `19`):**
  - **Bootstrap secrets** — the admin token hash and the OTLP export token —
    are read from the process environment at startup. custodian treats their
    source as opaque; `deed` owns the concrete store (SSM→tmpfs-env) and the
    read grant but never the runtime injection. custodian reads only
    `os.Getenv`.
  - **Third-party integration keys** (Steam, GitHub) live in custodian's own
    SQLite (this *revises* `10`'s original "env at startup" for integration
    keys), written through the authed admin API by `broom`, read at runtime, no
    restart on rotation, not AWS-specific. A self-hosted secrets manager was
    rejected as a 24/7 component that reintroduces a bootstrap token.
  - Either way, **no third-party key ever reaches `persona` or any browser** —
    the poller fetches server-side → SQLite → `persona` reads only
    `GET /v1/integrations/{source}`.
- **Rotation:** replace the secret and restart; revocation is implicit (old hash
  stops matching). No token registry, no self-credential endpoints, no grace
  period.
- **Leak detection:** admin-surface access logging (→ observability). Any
  `/admin/*` call the author did not make is the alarm.
- **Abuse, two layers:** nginx `limit_req` (per-IP, per-location, real-client-IP
  via CloudFront XFF, returns `429`) for origin hammering; **AWS WAF
  rate-based rule + Budgets alarm** for edge-cache wallet-DoS (chosen Option B —
  Budgets alone cannot throttle; WAF ~$6/mo converts a ~$1,678 attack month to
  ~$66). Both provisioned by `deed`. This pushes the total to ~£15/mo, a
  blessed breach of ADR-0001's ~£10 target.

### Observability (`11`)

- **One monitoring plane** (a hosted OTel-native backend on its free tier —
  Grafana Cloud), custodian instrumented **agentlessly via OpenTelemetry**. No
  CloudWatch (metered → budget-leak), no on-box Prometheus/Grafana (dies with
  the box), no agent sidecar, no separate uptime prober, no status page in v1.
- **Health** is a self-assessed `health` gauge (`1`/`0`) from `SELECT 1` +
  `HeadBucket` + disk/mem headroom, on a timer. **Third-party APIs excluded**
  from health. The gauge *is* the heartbeat.
- **Instrumentation:** OTel Go SDK, OTLP/HTTP export of metrics + traces + logs
  directly to a config-URL endpoint (basic auth). `journald` is only a
  short-retention local fallback buffer. Vendor-neutral by construction.
- **Alerting:** one rule — `health == 0` OR no-data-for-a-window → mobile push.
  Degraded and down are the same rule.
- **`/healthz`** is demoted to debug-only `curl`, not a load-bearing contract.
- Adds ~£0/mo (free tier, no payment method attached = the hard off-switch).

### Derived-data freshness (`12`)

- **One 5-minute poll tick**, both sources per tick, on the existing poller
  (the same loop that reaps stale media reservations). GitHub polled with
  `If-None-Match`. Interval is config, per-source overridable, 5-min default.
- **Scheduled polling, not lazy-on-miss** — predictable freshness, no
  first-visitor latency.
- **Edge/read TTL:** `Cache-Control: public, max-age=60, s-maxage=60,
  stale-while-revalidate=300` on `GET /v1/integrations/{source}`. Two clocks
  kept apart: the 5-min poll cadence (SQLite freshness) and the 60s edge TTL.
- **No manual override in v1** — derived data stays purely observed; a
  pin/suppress write is fog.
- **Idle = absence of a change**, not a rendered state. Idle polls don't insert
  and don't bump `updated_at`; `persona` sees `304`s. Startup poll guarantees a
  row; empty-array → `persona` hides the widget.
- **Custodian has zero opinion on data age** — always carries the fetch
  timestamp; `persona` owns the "recently active → stale" threshold. UI copy
  says "recently active", never "live"/"now".

### Persona coupling (context, not custodian work — `23`/`15`/`11`)

- Per `23`/`15`, `persona` is a **static SSG** (Astro, `output: 'static'`) with
  **no render tier**. It bakes blog prose, the blog index, and profile at build
  time by fetching custodian's **public `/v1`** (custodian is a *build-time*
  dependency for that content, off the reader's request path). A failed fetch
  fails the build loudly.
- **Derived widgets stay client-runtime fetch** against `/v1/integrations/*`
  (native custom elements, fetch on `connectedCallback`, hide on empty).
- This means the charting-era "blogs load at runtime via client fetch" is dead
  for *blogs*, but custodian's contract is unchanged — it still serves current
  content synchronously; whether the consumer reads it at build or at runtime is
  the consumer's choice. **Markdown rendering is not custodian's job** (it never
  was) — `persona` renders via remark/rehype at build; custodian stores raw
  GFM-core text.
- Custodian-down softens via CloudFront edge cache; a build-time blog snapshot
  backstop is deferred (it would reintroduce build-to-edit staleness).

## Testing Decisions

**What makes a good custodian test:** it asserts on **externally observable
behaviour** — HTTP status, response headers (`ETag`, `Cache-Control`,
`Access-Control-*`, `problem+json` bodies), response payloads, and the resulting
database state — never on internal function calls, private structs, or SQL
statement text. A test should read like a client of the API, because the API is
the contract every other component depends on.

**The single seam: the HTTP boundary.** Tests drive custodian black-box through
its two surfaces (`/v1/*` and `/admin/v1/*`) against the real chi server and a
**real in-process `modernc.org/sqlite`** database (temp file or `:memory:`), so
storage behaviour — uniqueness enforcement, slug freeze/move, append-on-change
timeseries, `updated_at` semantics — is exercised for real rather than mocked.
This is the highest available seam and keeps the test count of seams at one.

**The three outward edges are faked** (injected as interfaces at the seam),
because they leave the box and must not be hit in a test:

1. **S3** — the presign / `HEAD` / versioning surface used by the media flow.
   A fake lets tests exercise reserve → (simulated upload) → confirm → HEAD →
   `available`, and the stale-`pending` reaping, deterministically.
2. **The Steam / GitHub HTTP clients** — a fake lets tests drive the poller
   through changed / unchanged (`304`) / unreachable / idle / startup cases and
   assert the append-on-change and idle-is-no-row behaviour, plus that
   third-party failure never flips the `health` gauge.
3. **The OTLP exporter** — a fake/no-op sink lets tests assert the `health`
   gauge value and that it is emitted, without a live backend.

**Modules under test** (through the one seam): the log lifecycle
(create/patch/publish/rename/delete + all invariants and `problem+json` codes),
media reservation/confirm, profile upsert, the integration poller + read
freshness, auth on `/admin/*` (present/absent/wrong bearer), CORS and cache
headers on `/v1/*`, and the health/heartbeat gauge logic.

**Prior art:** there is no existing custodian test suite yet — this spec's
implementation establishes the pattern. It should follow idiomatic Go
`net/http/httptest` server tests: stand the real router up with
`httptest.NewServer` (or exercise the `http.Handler` directly), issue real
requests, and assert on responses + DB state. The generated OpenAPI Go client
(shared with `broom` per `13`) may be used as the test client so the tests also
exercise the published contract.

## Out of Scope

- **Building `broom`, `persona`, `blank`, or `deed`.** This spec is custodian
  only; where it touches them it records the *contract*, not their
  implementation.
- **Authoring or provisioning AWS infrastructure** — the EC2 box, S3 buckets,
  CloudFront distribution, WAF rate-based rule, Budgets alarm, IAM/instance
  profile, SSM parameters, and the OTLP-backend account are all `deed`/deploy
  work (`18`/`19`/`20`). custodian only *assumes* the instance-profile
  credential chain and reads its bootstrap secrets from the environment.
- **The OTLP/monitoring backend account setup** (dashboards, alert-rule wiring,
  IRM mobile push) — the operator configures the rented plane; custodian only
  exports OTLP to a config URL.
- **Version history for logs** (`log_revision`) — deferred, additive fog.
- **Manual override / pin for derived data** — fog; derived stays purely
  observed in v1.
- **A derived-data timeline / history read endpoint** — fog; the timeseries is
  stored so it is buildable later, but v1 serves only the latest row.
- **The `snippet` short-form type** — future feature, not v1.
- **LaTeX / math and any non-GFM-core body syntax** — explicitly not v1.
- **Media pipeline beyond raw upload** — optimisation, responsive variants,
  transform-on-ingest-vs-read, keeping originals alongside derivatives — waits
  on `blank`/persona rendering needs. Derivatives may slot under `<key>` but are
  not built.
- **custodian as a generic user-defined-type / BaaS layer** — future direction;
  v1 storage/API should merely not preclude it.
- **A public status page** — dropped for v1; folds into persona's page-inventory
  fog.
- **Migrating existing Notion content** into custodian — a separate one-off
  import task.
- **CloudFront↔origin perimeter security** (shared-secret header + origin TLS
  so the origin isn't reachable bypassing CloudFront) — a separate decision, kin
  to the WAF work, held out of this spec.

## Further Notes

- **Two clocks, stated once so nobody conflates them:** the 5-minute *poll
  cadence* governs how fresh the SQLite integration row is; the 60-second *edge
  TTL* governs how long CloudFront/browsers hold a read. They are independent by
  design.
- **The invariant chain behind trustworthy media URLs:** key uniqueness enforced
  at reserve → `HEAD` verification at confirm → only-then `available`. This is
  what lets `persona` paste bare CDN URLs and lets `broom`'s pre-delete
  reference scan be a courtesy rather than the safety net (S3 versioning is the
  real net).
- **Why profile lives in custodian at all:** originally justified by freshness
  (`03`), but `23`'s reintroduction of rebuild-to-edit for baked content killed
  that rationale. Per `15` the *live* rationale is **retrieval** — a future QnA
  bot answers from the profile via the same public API — plus keeping all
  content in one owned store. Do not re-derive the freshness argument.
- **Open cost caveat (carried, non-blocking):** confirm the 30-day-retention S3
  cost (Standard storage + versioning non-current copies + WAL segment history)
  against ADR-0001's envelope when `deed` configures the buckets. Near-certainly
  pennies at blog scale.
- **Budget reality:** with WAF, the steady-state total is ~£15/mo, a blessed,
  recorded breach of the ~£10 target for a hard ceiling on catastrophic spend.
  Observability adds ~£0.
- The full monthly cost model + Budgets/WAF provisioning is `deed`'s build-phase
  work, not custodian's.
