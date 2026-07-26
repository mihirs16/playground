# Authored-content domain model

Type: grilling
Status: resolved

## Question

What is the shape of an authored piece of content, in domain terms, before any storage or transport decision is made?

- **Blog posts**: what frontmatter fields exist, and which are required? (title, slug, publish date, updated date, tags, description/excerpt, cover image, draft flag, reading time?)
- **Slugs**: author-chosen or derived from the title? Immutable once published? What happens when a title changes?
- **Drafts**: does custodian hold unpublished work at all, or is publishing an act that happens elsewhere? Can persona request drafts with auth (a preview mode)?
- **Media references**: how does a markdown body point at an uploaded image? An opaque id resolved by custodian, a stable public URL, or a relative path? This determines whether markdown is portable out of custodian.
- **Markdown flavour**: plain CommonMark, GFM, or extended with custom directives? Does the highlight-yellow emphasis from `blank` need syntax (e.g. `==mark==`) or is it just `<strong>` styled?
- **Profile content**: is it genuinely "just another authored type" as the charting session assumed, or does about / experience / skills / resume-link need bespoke shapes?
- **Curated projects**: field shape for the authored showcase, now that it's a separate concept from GitHub activity.
- **Ubiquitous language**: settle the terms themselves — is it a "post", an "entry", a "note"? Does "content" mean all three buckets or only authored?

## Context

Use `/domain-modeling` alongside `/grilling` — output should update `CONTEXT.md` with the agreed vocabulary, since every later ticket and all four specs will use these terms.

The deprecated repo's content shapes are known and worth reusing where they still fit: `experience` had order / company / role / dateRange / details (pipe-separated in one field — worth fixing), `projects` had order / name / description / tools / url, `skills` was a category-to-list table, `about` was a single rich-text block, and the resume link was smuggled inside a Notion code block. There were no blogs and no media at all, so those are genuinely new.

Deliberately upstream of the storage decision: decide what the things *are* before deciding where they sit.

## Blocks

`08` storage model, `09` API contract

## Answer

Full ubiquitous language recorded in `CONTEXT.md` (§ "The authored-content domain model"). Summary:

**"Content" spans all three buckets** (authored, derived, profile), not just authored.

**`log`** — the one first-class authored-structured type; long-form/blog-like.
- Fields: title, subtitle, slug, cover image, reading time, tags, created date, updated date, body, state. No description/excerpt yet (deferred to blog-list UI).
- State: `listed`/`unlisted`. Always link-reachable by slug; state only controls index visibility. `unlisted` = both never-published drafts *and* taken-down logs (no separate un-shareable state). Draft is shareable by link before first listing. Unpublish allowed.
- Slug: author-chosen; **frozen while listed, mutable while unlisted**.
- Body: **GFM core** (CommonMark + tables, strikethrough, task lists, autolinks). No footnotes (portability footgun). No novel syntax. Highlight-yellow = plain `**strong**`. Blockquote (`>`) carries both quotations and margin asides; `persona` chooses the side, author gets no left/right control.

**`snippet`** — short-form/note-like. **Future feature, not v1**; likely a summary derived from a log.

**`media`** — first-class, independent type (not log-owned). CRUD + list + search. Fields: url, filename, content-type, size, upload date, caption, attribution. URL is generic public domain-owned CDN (`cdn.mihirsingh.dev/…`), not log-scoped, so bodies stay portable. Referenced only as URL text (no referential integrity); `broom` reference-check searches log bodies before delete, else orphaning is the author's responsibility.

**`profile`** — table of keyed records, one row per key (`id` + opaque JSON `body`): `about` (markdown body), `experience`, `skills`, `resume-link`, `curated-projects`. `custodian` does not validate the body; shape is a persona↔custodian convention. No-guardrail trade accepted (low-churn, single-author).

**`integration`** — derived/third-party type; one record per source (Steam, GitHub) with opaque JSON `body` + fetch timestamp. Mirrors `profile`'s id+body shape.

**Scope decision:** the Notion-style user-defined-type / BaaS "platform" idea was explicitly ruled a **future direction, not v1** — same "capable-of, don't-build" framing as `blank`/`deed`. Destination stays scoped to explicit first-class types.

**Graduated to fog:** `snippet`; LaTeX/math in log bodies; description/excerpt field (pending blog-list UI); custodian-as-generic-BaaS meta-type layer.
