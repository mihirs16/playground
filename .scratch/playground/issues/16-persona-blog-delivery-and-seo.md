# persona: blog delivery, crawlability and permalinks

Type: grilling
Status: resolved
Blocked by: 09, 15

## Question

Blogs load from custodian at runtime. What does a crawler, a link preview scraper, and a reader with a direct permalink actually get?

- **Crawlability** — client-fetched blog bodies mean the static HTML is an empty shell. Search engines execute JavaScript inconsistently and with delay; most link-preview scrapers (Slack, iMessage, WhatsApp, Mastodon, LinkedIn) execute none at all. A shared blog post would preview as a blank page. Is that acceptable, or does it need solving?
- **Per-post OG tags** — these must be in the initial HTML response to work at all. Purely client-side rendering cannot produce them. If per-post previews matter, something has to pre-render or inject them.
- **Permalinks** — does `/blog/some-post` resolve as a real URL on a static host, or only via client-side routing from the index? Direct navigation and hard refresh have to work.
- **Options if crawlability does matter**: build-time prerender of published posts with a runtime refresh for edits; an edge worker that fetches from custodian and injects meta tags; ISR-style regeneration on a webhook from custodian; or serving persona from a small server after all.
- **Empty and error states** — what renders when custodian is slow or down? A visitor arriving at a blog permalink during an outage sees nothing, unlike a static site.
- **RSS/Atom feed** — generated where? A feed is build-time by nature, which sits awkwardly with runtime-loaded posts.
- **Does any of this change the runtime-loading decision**, or is it accepted with mitigations?

## Context

Blocked on `09` (the read contract) and `15` (the framework determines which mitigations are even available).

This ticket exists because the charting session flagged a tension rather than resolving it. You chose runtime loading for blogs with the reasoning that custodian needs health monitoring regardless — which addresses availability, but not crawlability or link previews. Those are separate failure modes and neither is fixed by monitoring.

Stated plainly so a later session doesn't have to rediscover it: markdown blogs are the content type where crawlability and share previews matter most, and runtime loading is the delivery model that serves them worst. That may still be the right trade for a personal site — but it should be a decision, not an accident. `11` covers the availability half.

## Update — most of this ticket's premise is now settled upstream

`23` (blog delivery model) and `15` (framework) landed after this ticket was written and **overtake its "blogs load at runtime" premise**:

- **Delivery = build-time SSG (`23`), framework = Astro static (`15`).** Blog prose is baked into the initial HTML payload, so crawlability, link-preview scrapers, and per-post OG tags are all satisfied by construction — the empty-shell failure mode is gone. custodian is a build-time (not reader-path) dependency, so an outage no longer blanks a permalink.
- **Permalinks (`15`)** — real static files at `/logs/<slug>/` via `getStaticPaths()`, clean URLs via `build.format: 'directory'` + a `deed`-owned CloudFront Function index-rewrite. No client router. Direct nav / hard refresh work.
- **Empty/error states** — only the **derived widgets** fetch at runtime now (`12`/`14`); each hides itself on empty. Blogs never fetch at runtime.

**What genuinely remains for this session to bank:**

1. **Markdown-pipeline ownership + the fate of `<blank-markdown>`** (surfaced by `15`). Under SSG the build-time renderer **must** be Astro's remark/rehype — it *cannot* be `<blank-markdown>` (Lit-at-build = `@lit-labs/ssr`, banned by `21`). Owner's steer: **`<blank-markdown>` leaves `blank`.** Bank one of: **(a)** persona-owned runtime-markdown Web Component, or **(b)** dissolves into Astro's build, no markdown component in v1 (leaning **b**). **Either way this revises `14`.** Also confirm raw-HTML passthrough (`rehype-raw`) so logs can embed live `blank` components (safe: single-trusted-author, GFM-core-legal).
2. **Per-post OG/meta tags** — now trivially build-time in the Astro page head, but the exact set (title/description/image source) still needs stating; interacts with `03`'s deferred `description`/excerpt fog.
3. **RSS/Atom feed** — build-time generation over the same baked slug set; confirm in/out for v1.

The crawlability-vs-runtime *trade* this ticket was created to force is **resolved (SSG won, `23`)**; what's left is the residual build-time detail above.

## Answer

The crawlability / link-preview / permalink / outage failure modes are **all resolved upstream** by `23` (build-time SSG) + `15` (Astro static): blog prose is baked into the initial HTML payload, custodian is a build-time-only (not reader-path) dependency, and permalinks are real static files. This session banks only the residual build-time detail.

1. **`<blank-markdown>` — dropped from v1 (option b).** Under SSG the build-time markdown renderer *must* be Astro's remark/rehype pipeline; a Lit component at build would drag in `@lit-labs/ssr`, banned by `21`. Its only consumer was the blog path, which SSG removed, so a persona-owned component (option a) would ship with zero callers. Markdown-to-HTML **dissolves into Astro's build step** — a build tool, not a component; the custom-element machinery buys nothing when it only ever runs at build. Custodian still stores raw markdown; only the rendering location moves. A runtime-markdown component graduates from fog later if a real second surface (live preview, comment box) ever needs one. **Revises `14`** (drops `<blank-markdown>` from the inventory).

2. **`rehype-raw` enabled — documented risk.** Raw HTML in a log body passes through to output (not escaped/stripped) so a log can embed live `blank` components (e.g. a post about `blank` showing a real `<blank-stat>`). This is normally an XSS vector; it is safe here *only* because of the **single-trusted-author** invariant (and raw HTML is GFM-core-legal). That assumption is **documented in both the spec and an inline comment at the pipeline config**, so the caveat travels with the code and does not rot into an unexamined hole if authorship ever changes.

3. **OG / meta tags — build-time in the Astro page head.** Set: **title**, an **optional `description`**, a **single site-wide default OG image** (monotone card; no per-post image field — that stays fog, being the classic never-populated field), plus standard boilerplate (canonical URL, `og:type=article`, `og:site_name`, `twitter:card=summary_large_image`). A log with no `description` simply omits the tag — an absent preview beats a mangled auto-truncated one.

4. **New `description` field graduates from `03` fog.** `subtitle` is **editorial** (a preface line or quote, part of the reading experience) and must not be conscripted as an SEO summary. So OG description gets its own source: a dedicated **plain-prose, optional `description`** on the log, also reusable as a blog-list excerpt later. **Revises `03`** (un-defers the description/excerpt field), **`08`** (storage column), **`09`** (API field), **`17`** (`broom logs new` metadata prompt).

5. **RSS 2.0 feed — in v1.** Build-time `/rss.xml` via `@astrojs/rss` over the same `getStaticPaths()` slug set — no new data source, no runtime tier. RSS 2.0 over Atom for widest reader support; the correctness edge doesn't matter at this scale. Reuses title / `description` (absent → fine) / permalink.
