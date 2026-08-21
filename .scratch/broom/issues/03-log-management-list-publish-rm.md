# 03 — Log management: `list`, `publish`/`unpublish`, `rm`

**What to build:** The commands that manage posts once they exist.
`broom logs list [--listed|--unlisted]` shows the author's posts of any state —
including the `unlisted` drafts the public index hides — so work in progress can
be found and managed. `broom logs publish <slug>` and `broom logs unpublish
<slug>` toggle a post's state; on the wire both are the same
`PATCH /admin/v1/logs/{slug}` that carries the body edit (publish → `listed`,
unpublish → `unlisted`), presented as distinct verbs — no bespoke publish
endpoints. `broom logs rm <slug>` deletes a post so a draft or retired post can be
removed entirely. An attempt to rename a `listed` post's slug is reported clearly
from custodian's `slug_frozen_while_listed` code, so the author understands that
published links are deliberately frozen.

**Blocked by:** 02.

**Status:** done

- [x] `logs list` shows posts of any state; `--listed`/`--unlisted` filter accordingly, including drafts hidden from the public index
- [x] `logs publish`/`unpublish` toggle state via `PATCH {state}` (listed/unlisted), no bespoke endpoint
- [x] `logs rm <slug>` deletes a post
- [x] `slug_frozen_while_listed` is surfaced as a clear "published links are frozen" message
- [x] All three verbs exercised through the fake custodian, asserting method/path/body of each request
