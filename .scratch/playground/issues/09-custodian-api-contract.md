# custodian: API contract

Type: grilling
Status: resolved
Blocked by: 03, 08

## Question

What is custodian's API contract — the shared boundary that both the cli and persona depend on?

- **Style** — REST, GraphQL, or RPC? For one known writer and one known reader, GraphQL's flexibility mostly buys nothing; make the case either way rather than assuming.
- **Contract format** — OpenAPI spec, or hand-written? This decides whether the cli and persona get generated clients, which matters more if they're in different languages.
- **Read path** — what persona calls, and with what caching semantics. Blogs load at runtime, so list-and-detail shapes, pagination, `ETag`/`Cache-Control`, and CORS all matter concretely.
- **Write path** — what the cli calls. How does media upload work: direct multipart to custodian, or a presigned URL straight to object storage? The latter keeps custodian off the upload path entirely.
- **Are the two paths one API or two?** A public read API and a private admin API have genuinely different shapes, auth, and cache behaviour. Splitting them is a real option.
- **Errors** — a consistent error shape, and what the cli surfaces to you when something fails.
- **Versioning** — how the contract evolves once persona is deployed against it.

## Context

Blocked on `03` (needs the domain model's entities and vocabulary) and `08` (needs to know whether writes are fast or commit-shaped, and whether media goes through custodian or around it).

This is the highest-fan-out ticket on the map — the cli spec, persona's data layer, and the freshness strategy all depend on it. Worth more care than its neighbours.

Consider prototyping the contract as a written spec artifact and linking it, rather than settling it purely in conversation. A concrete request/response sketch for "fetch a blog post" and "publish a blog post with two images" will expose problems that abstract discussion won't.

## Blocks

`10` auth model, `12` freshness and caching, `16` persona blog delivery, `17` cli language

## Answer

Ten decisions, walked as a dependency tree from the root (style) outward. Each was put to the human one at a time; recommendations and the chosen answers are recorded below. **No standalone spec artifact was written** — the concrete flows at the bottom did their problem-exposing job in the grilling itself, and the full OpenAPI `.yaml` is a destination deliverable (the custodian spec) that also waits on `10`'s auth decisions, so authoring it now would be a premature, half-built pull toward the edge of the map. The sketches are recorded here as the *decision record's* illustration, not as the beginning of the build.

### The ten decisions

1. **Style — REST/JSON.** GraphQL's flexibility buys nothing with one controlled reader and weakens the HTTP caching `12` lives on; RPC (Connect/gRPC) fights CloudFront edge-caching and same-origin browser GETs. REST's `GET` + `ETag` + `Cache-Control` is exactly persona's runtime-blog machinery.
2. **Topology — two surfaces, one Go binary, split by *audience* not verb.** A **public** surface (`/v1/*`: unauth, GET-only, cacheable, CORS) and an **admin** surface (`/admin/v1/*`: authenticated, full CRUD, `no-store`, no CORS). Split matters because `broom` *reads* too (unlisted drafts, media search) — so the seam is public-vs-admin, not read-vs-write. Path-prefix split makes CloudFront behaviours trivial (cache-all vs cache-none by path) and means `10`'s auth only ever guards `/admin/*`. Stays one process on the `t4g.micro`.
3. **Contract format — OpenAPI 3.1, spec-first.** The hand-authored `.yaml` is source of truth; Go server interfaces (oapi-codegen) and the TS client (openapi-typescript) both generate from it, so the compiler catches drift across the Go↔TS boundary. Code-first (swaggo annotations) rejected as drift-prone and lower-fidelity. Publishable contract = same forcing-function discipline as `blank`'s public semver.
4. **Read path.**
   - **Index** `GET /v1/logs?limit=&offset=&tag=` → `{ total, items:[summary] }`; **summaries omit `body`**, listed-only. **offset/limit** pagination (cursor is overkill at tens-of-posts; each page is its own cacheable URL; no contract change as it grows).
   - **Detail** `GET /v1/logs/{slug}` → full log incl. `body`, **any state** (unlisted is link-reachable = still public).
   - Also `GET /v1/integrations/{source}`, `GET /v1/profile/{key}`.
   - **persona never fetches media from the API** — media bytes come straight from `cdn.mihirsingh.dev/<key>`; media metadata is admin-only.
   - **Caching mechanism (numbers are `12`'s):** every read carries **`ETag` + `Cache-Control`**, custodian honours **`If-None-Match` → `304`**. ETag inputs: `updated_at` (logs, also a free `Last-Modified`) / fetch timestamp (integrations). Per-type `Cache-Control` (logs/profile long+revalidate, integrations short); CloudFront honours `s-maxage`. Mirrors downstream to persona the same free-`304` cheapness `02` found GitHub gives custodian upstream.
   - **CORS — explicit origin allowlist** (not wildcard), exposing `ETag` and allowing `If-None-Match`; allowlist lives in **static deploy config** (env/file + reload; deed-adjacent infra posture), *not* a `broom`-managed settings resource. Admin surface: no CORS.
5. **Media upload — presigned S3 `PUT`, custodian off the byte path.** `POST /admin/v1/media` reserves a `pending` record (key uniqueness enforced *now*, per `08`) and returns `{ upload_url (presigned), url (public cdn), expires_at }`; `broom` `PUT`s bytes straight to S3; **`POST /admin/v1/media/{key}/confirm`** → custodian **`HEAD`s S3** (object exists + content-type/size match) and only then flips to `available`. Invariant: *every `available` media record has real bytes behind it*, which is what makes the public CDN URLs and `broom`'s pre-delete reference-check (from `08`) trustworthy. Stale `pending` records past `expires_at` are reaped by custodian's **existing poller** (the Steam/GitHub loop from `02`), so no new background machinery. `broom` needs no AWS credentials — the presigned URL carries scoped, time-limited auth.
6. **Log identity — slug is sole identity** (human's call over the recommended surrogate-id). Slug is author-chosen, **mutable while unlisted, frozen once listed**. A rename is a **server-performed move**: `PATCH /admin/v1/logs/{slug}` with a new `slug` → custodian relocates the record if unlisted and the new slug is unique, and echoes the new slug back for `broom` to use. custodian enforces the freeze-while-listed and uniqueness rules.
7. **Log write verbs — `POST` + `PATCH` + `DELETE`.** Create via `POST /admin/v1/logs` (slug in body, 409 if taken); partial edits via `PATCH` where `state`/`slug`/`body` are ordinary fields (publish = `PATCH {state:"listed"}`); delete via `DELETE`. Invariants enforced server-side and documented on the one endpoint by the spec. No full-replace `PUT` (would force resending `body` to toggle state), no explicit `:list`/`:unlist` actions.
8. **Profile & integrations.** Profile: `PUT /admin/v1/profile/{key}` — idempotent upsert of the opaque JSON body for a known key, custodian doing **no** shape validation (per `03`). Integrations: bodies are **poller-fetched, never authored**; the only integration write is **`POST /admin/v1/integrations/{source}/refresh`**, which forces an immediate poll and returns the fresh record (valuable during setup/debugging — after rotating a Steam key or fixing a GitHub PAT).
9. **Errors — RFC 9457 Problem Details** (`application/problem+json`): `type`/`title`/`status`/`detail`/`instance` + a **stable `code` extension** (`slug_conflict`, `media_key_taken`, `log_not_found`, `slug_frozen_while_listed`, …) + a **field-errors array** for validation. `broom` prints `detail`, branches on `code` (e.g. offer a new slug on `slug_conflict`), lists field errors. Typed in the generated clients because it's in the spec.
10. **Versioning — URL path `/v1` + additive-only within a major + OpenAPI `info.version` semver.** Add fields/endpoints freely; never remove/rename/repurpose within a major; a breaking change is `/v2` living beside `/v1` during cutover. Path versioning matches CloudFront's path cache-keys and lets a future break run side-by-side rather than a flag-day, keeping the `blank`-style "boundary is real" discipline.

### Endpoint surface

- **Public** — `GET /v1/logs`, `GET /v1/logs/{slug}`, `GET /v1/integrations/{source}`, `GET /v1/profile/{key}`
- **Admin** — `GET|POST /admin/v1/logs`, `PATCH|DELETE /admin/v1/logs/{slug}` · `GET|POST /admin/v1/media`, `GET|DELETE /admin/v1/media/{key}`, `POST /admin/v1/media/{key}/confirm` · `PUT /admin/v1/profile/{key}` · `POST /admin/v1/integrations/{source}/refresh`

### Draft sketch — Flow A: persona fetches a blog post at runtime

```http
GET /v1/logs/lakeside-weekend HTTP/1.1
If-None-Match: "9f2a-17c"
Origin: https://mihirsingh.dev
```
```http
200 OK
ETag: "e4b1-201"
Cache-Control: public, max-age=<12>, stale-while-revalidate=<12>
Access-Control-Allow-Origin: https://mihirsingh.dev
Access-Control-Expose-Headers: ETag
Vary: Origin
Content-Type: application/json

{ "slug":"lakeside-weekend", "title":"lakeside weekend", "subtitle":"…",
  "cover_image":"https://cdn.mihirsingh.dev/sunset-over-lake",
  "reading_time":4, "tags":["travel"], "state":"listed",
  "created_at":"2026-07-20T…Z", "updated_at":"2026-07-23T…Z",
  "body":"# lakeside weekend\n\n![sunset](https://cdn.mihirsingh.dev/sunset-over-lake)…" }
```
Unchanged since last fetch → `304 Not Modified`, empty body. Index: `GET /v1/logs?limit=20&offset=0&tag=travel` → `{ "total":7, "items":[ …summaries, no body… ] }`.

### Draft sketch — Flow B: broom publishes a post with two images

```http
# ×2, once per image
POST /admin/v1/media          { "key":"sunset-over-lake", "content_type":"image/webp", "size":204831, "filename":"sunset.webp", "caption":"…" }
→ 201  { "key":"sunset-over-lake", "state":"pending", "upload_url":"https://…s3…?X-Amz-Signature=…", "url":"https://cdn.mihirsingh.dev/sunset-over-lake", "expires_at":"…" }
PUT  <upload_url>             (bytes straight to S3)                          → 200
POST /admin/v1/media/sunset-over-lake/confirm   → 200 { "state":"available", … }

# then the log — drafted unlisted, body references the CDN URLs as plain markdown
POST  /admin/v1/logs         { "slug":"lakeside-weekend", "title":"…", "cover_image":"https://cdn.mihirsingh.dev/sunset-over-lake", "body":"…![sunset](https://cdn.mihirsingh.dev/sunset-over-lake)…", "state":"unlisted" }
→ 201  { "slug":"lakeside-weekend", "state":"unlisted", … }

# publish → slug freezes
PATCH /admin/v1/logs/lakeside-weekend   { "state":"listed" }   → 200 { "state":"listed", … }
```
Slug collision on create:
```http
409 Conflict — application/problem+json
{ "type":"https://custodian.mihirsingh.dev/errors/slug-conflict", "title":"Slug already in use",
  "status":409, "code":"slug_conflict", "detail":"A log with slug 'lakeside-weekend' already exists." }
```

### Ripples

- **Unblocks the frontier:** `10` (auth model — blocked by `09` only), `12` (freshness/caching — `02` already resolved), `17` (cli language — `06`/`07` already resolved) all join the frontier. `16` (persona blog delivery) stays **blocked** by `15` (persona framework), not by `09`.
- **Boundary handed to `12`:** `09` fixed the caching *mechanism* (`ETag`/`Cache-Control`/`304`, per-type header presence); `12` owns the freshness *policy* — the actual TTL seconds, especially for the deliberately-fresh integrations.
- **Boundary handed to `10`:** auth applies only to `/admin/*`; the credential rides a **header, never a cookie** (this is what keeps the public surface's CORS allowlist free of credentialed-request concerns). The exact scheme is `10`'s to decide.
- **Fed to `17`:** `broom` is a pure OpenAPI-generated client of the admin surface — no AWS credentials (presigned URLs carry their own auth), switches on `problem+json` `code`s, tracks slugs (not ids).
- **No new tickets surfaced.** The full OpenAPI `.yaml` authoring belongs to the custodian spec document under the "five spec documents" fog; media list/search shapes and body-stored-verbatim (no GFM validation in custodian — rendering is persona/`blank`'s job via goldmark) are custodian-spec detail, not fresh decisions.
