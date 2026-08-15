# 02 — Authoring core: `logs new` and `logs edit`

**What to build:** The authoring loop — the highest-weight slice, since a poor
authoring UX is the one thing most likely to get broom abandoned. `broom logs new`
interactively prompts for metadata (title, subtitle, tags, optional plain-prose
`description`), then creates the post *immediately* as an `unlisted` draft the
instant metadata is entered — the slug-is-the-post invariant, so there is never a
post the author has started that custodian does not know about. It then launches
`$EDITOR` on the initially empty body and `PATCH`es the body to custodian on save
and close. An empty body is valid, including aborting the editor without writing —
a freshly created post is a valid empty `unlisted` draft, not an error.
`broom logs edit <slug>` round-trips an existing post identically: pull the
current body into a temp file, open `$EDITOR`, `PATCH` on save. A slug collision
on create is surfaced legibly from custodian's `slug_conflict` code with a prompt
to choose another slug, so the author recovers in-flow rather than reading a raw
HTTP error. This slice establishes the `$EDITOR` seam (a scripted fake in tests
that writes known content or exits without writing).

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] `logs new` prompts metadata and creates an `unlisted` post immediately, before the editor opens
- [ ] `logs new` launches `$EDITOR` on the empty body and `PATCH`es on save
- [ ] Empty body and abort-without-writing both leave a valid empty `unlisted` draft
- [ ] `logs edit <slug>` pulls the body into a temp file, opens `$EDITOR`, and `PATCH`es on save
- [ ] `slug_conflict` on create is surfaced legibly with a prompt to choose another slug
- [ ] `$EDITOR` seam is faked in tests (writes known content / exits without writing) and the full `logs new` and `logs edit` loops run through the fake custodian
