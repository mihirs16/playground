# 05 — Profile read + write

**What to build:** persona can bake profile content and broom can edit it.
`GET /v1/profile/{key}` serves a profile record by key (`about`, `experience`,
`skills`, `resume-link`, `curated-projects`) on the public surface, carrying the
same `ETag` + long revalidate-friendly `Cache-Control` treatment and
`If-None-Match` → `304` handling as logs. `PUT /admin/v1/profile/{key}` upserts a
profile record with an opaque JSON `body` that custodian does not validate — the
shape is controlled by convention between broom and persona. This ticket
introduces the `profile` table (one row per key: an `id` plus the opaque JSON
`body`).

**Blocked by:** 02, 03.

**Status:** ready-for-agent

- [ ] `profile` table stores one row per key with an opaque JSON body
- [ ] `GET /v1/profile/{key}` serves the record with `ETag` + `Cache-Control` and honours `If-None-Match` → `304`
- [ ] `PUT /admin/v1/profile/{key}` upserts and does not validate the body shape
- [ ] Admin write is authed + `no-store`; public read is on the CORS allowlist
- [ ] Tests assert upsert semantics and read headers black-box against real sqlite
