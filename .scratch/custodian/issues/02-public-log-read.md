# 02 — Public log read (index + detail) with caching & CORS

**What to build:** persona's build can bake the blog. `GET /v1/logs` returns a
paginated index `{ total, items }` where each item is a summary that omits
`body`, includes only `listed` logs, accepts `limit`/`offset` and an optional
`tag` filter, and returns `total` alongside the items for pagination controls.
`GET /v1/logs/{slug}` returns a single log including its full `body` in any
state, so an `unlisted` draft is previewable at its real URL. This ticket
introduces the `log` table (title, subtitle, slug, cover image, reading time,
tags, created_at, updated_at, body, state, optional `description`). Every
response carries an `ETag` (from `updated_at`) and a per-type `Cache-Control`
(logs long + revalidate); custodian honours `If-None-Match` and returns `304 Not
Modified` with an empty body. The public surface applies an explicit CORS origin
allowlist (never wildcard), exposes `ETag`, and allows `If-None-Match`.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] `GET /v1/logs` returns `{ total, items }`, summaries omit `body`, listed-only
- [ ] Index honours `limit`/`offset` and optional `tag` filter
- [ ] `GET /v1/logs/{slug}` returns full body for any state, including `unlisted`
- [ ] Every read carries `ETag` + long revalidate-friendly `Cache-Control`
- [ ] `If-None-Match` match returns `304` with empty body
- [ ] CORS allowlist is explicit (not wildcard), exposes `ETag`, allows `If-None-Match`
- [ ] Tests drive both endpoints black-box against real sqlite and assert payloads + headers + DB state
