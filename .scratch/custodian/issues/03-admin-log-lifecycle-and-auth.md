# 03 — Admin log lifecycle + auth + problem+json

**What to build:** broom can author and manage posts over the admin surface.
Auth middleware guards `/admin/*` only: a single long-lived bearer token on
`Authorization: Bearer`, of which custodian stores only the hash (read from the
process environment at startup); present/absent/wrong-token cases behave
correctly, and the credential never rides a cookie. Admin responses are
`no-store`. `POST /admin/v1/logs` creates a log with an author-chosen slug as an
`unlisted` draft; a slug collision returns `409` with a stable `slug_conflict`
code. `PATCH /admin/v1/logs/{slug}` partially updates — `state`, `slug`, `body`,
`description`, and metadata are ordinary fields — so publishing is
`PATCH {state:"listed"}` and unpublishing is `PATCH {state:"unlisted"}` with no
bespoke endpoints. A slug rename via `PATCH` is a server-performed move that
echoes the new slug back while the log is `unlisted`, and is rejected with
`slug_frozen_while_listed` once `listed`. `DELETE` removes a post. Admin `GET`
lists logs of any state and fetches any log including unlisted drafts. Every
error is RFC 9457 `application/problem+json` with a stable `code` and, for
validation failures, a field-errors array.

**Blocked by:** 02.

**Status:** done

- [ ] `/admin/*` requires a valid hashed bearer; absent/wrong → problem+json auth error; token hash from env
- [ ] Admin responses are `no-store`; credential accepted only via header, never cookie
- [ ] `POST` creates an `unlisted` draft; slug collision → `409 slug_conflict`
- [ ] `PATCH` partially updates; `state` transitions publish/unpublish with no bespoke endpoints
- [ ] Slug rename while `unlisted` performs the move and echoes the new slug; rename while `listed` → `slug_frozen_while_listed`
- [ ] `DELETE` removes a log; admin `GET` lists/fetches any state incl. unlisted
- [ ] All errors are RFC 9457 `application/problem+json` with stable `code` + field-errors array for validation
