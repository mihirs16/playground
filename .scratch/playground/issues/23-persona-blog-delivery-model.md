# persona: blog delivery model — runtime fetch vs build-time SSG / ISR

Type: grilling
Status: resolved
Blocked by: 21

## Question

The charting decision was **"persona bakes profile at build time; blogs, Steam and GitHub load at *runtime* from custodian"** — blogs delivered by client-side fetch. Resolving `21` (rendering foundation) exposed that this fights the blog's own requirements:

- **SEO + paint need prose in the server payload.** `21` settled that `<blank-markdown>` prose renders to **light-DOM semantic HTML in the initial response** (the whole reason it's the light-DOM exception). Pure client-runtime-fetch delivers an empty shell to crawlers and to first paint, then hydrates — the exact anti-pattern `21`'s light-DOM decision exists to avoid. Runtime client-fetch and "prose in the payload" are in direct contradiction for the blog body.
- **But runtime was chosen for a real reason** — the core motivation: escape the deprecated site's `getStaticProps`-with-no-revalidation **rebuild-to-edit** loop. A naive build-time SSG reintroduces exactly that: publish a log in custodian → nothing changes on persona until a full rebuild. `11` already deferred a "build-time blog snapshot" on these grounds.

The decision to make: **how are blogs *delivered*** such that prose is server-rendered (crawlable, fast-painting) **without** reintroducing rebuild-to-edit? Candidate shapes to weigh:

- **ISR / on-demand revalidation** — static pages served from the edge, revalidated on a timer and/or via a **custodian → persona webhook** on publish (`broom`/admin write triggers a rebuild-of-one-path). Static delivery *and* near-instant edits.
- **Runtime server-render (SSR)** — persona is a running server that fetches custodian per request and renders prose server-side. Fresh always, but adds a long-lived render tier (cost, ops — cuts against `01`'s posture) and puts custodian on the request path for every page.
- **Pure build-time SSG** — simplest/cheapest, but the rebuild-to-edit loop returns (the thing custodian exists to kill).
- **Hybrid** — profile baked at build (already decided), blog *index/prose* via ISR, derived widgets (Steam/GitHub) stay client-runtime (they're decorative, not SEO/paint-critical — `12`/`14` already say persona hides an empty widget).

**Invariant to hold regardless:** blogs stay authored into **custodian's SQLite** as the single source of truth / indexing store (`08`) — this ticket is about *delivery*, not storage. Custodian remains canonical; the question is what sits between it and the reader.

## Context

Surfaced while resolving `21`. Revises the charting decision ("blogs load at runtime") and revisits `11`'s deferred build-time-snapshot. Tightly coupled to `15` (persona framework — the viable framework set depends on whether persona needs ISR/on-demand-revalidation vs pure SSG vs SSR) and feeds `16` (blog delivery + SEO). Weigh every option against `01`'s "self-manage compute, rent durability, ~£10-15/mo, no needless long-lived tiers" posture and the core motivation (no rebuild-to-edit).

## Blocks

`15` persona framework — a framework can't be chosen until the delivery model (SSG / ISR / SSR / hybrid) it must support is fixed.

## Answer

**Blogs are build-time SSG. A hybrid: static prose + profile baked at build, decorative derived data fetched client-side at runtime.** No long-lived render tier for persona — it stays static objects on S3/CloudFront exactly as `01`/`19` settled.

Decision walked in order:

1. **No running render tier for persona.** SSR-on-a-server (candidate "runtime server-render") is ruled out by construction — persona stays a static-object host. This keeps `01`'s "no needless long-lived tiers" posture intact and keeps custodian off the per-request reader path. So "ISR" only ever meant "regenerate static objects on a trigger and re-sync to S3 + invalidate CloudFront" — never Next-on-a-server.

2. **SEO is general, not Google-only — so prose *must* be in the initial HTML payload.** Google renders JS (evergreen-Chromium WRS) so pure client-side render (CSR) *could* be indexed by Google alone, but non-Google crawlers and social/link unfurlers (Bing, DuckDuckGo, Slack, LinkedIn, iMessage, X) largely do **not** execute JS. That kills pure CSR and kills the original charting decision's "blogs load at runtime via client fetch." Prose-in-payload is now a hard invariant — the definition of SSR here.

3. **Prose-in-payload splits only by *when/where* the render happens.** Live contenders that satisfy all crawlers *and* avoid a long-lived tier *and* keep presentation out of custodian: **(1) prerender-to-static (SSG)** — render ahead of time, freshness via a regeneration trigger; **(2) serverless/edge SSR** — render on cache-miss in a Lambda, freshness via CloudFront TTL. (Custodian-SSR-at-origin rejected: drags presentation into custodian, puts it on the reader path, makes blog-down = custodian-down. Long-lived persona server rejected per step 1.)

4. **Chosen: SSG (option 1).** Blog cadence is low (not writing daily), so second-instant freshness buys nothing and isn't worth serverless SSR's extra moving parts / cold-starts. Weighted "minimum moving parts / it's just files" over "fresh by default."

5. **Trigger is manual.** A persona rebuild is a command *you* run (a root `just` recipe / deploy per `13`/`19`) when you want a batch of posts live — **no custodian→persona webhook, no timer.** broom and custodian stay completely ignorant of persona; the coupling is one-directional and build-time only.

**This deliberately reintroduces rebuild-to-edit for blogs** — a log published in custodian is invisible on persona until the next manual rebuild — and that reverses the map's stated core motivation *for blogs specifically*. Accepted consciously and reframed as **editorial batching** (author controls when a batch goes live), not a defeat. Custodian still fully earns its place: single source of truth, structured store, read/admin API, and the broom authoring flow (write without git-commit-per-post or a code deploy). The *freshness* half of custodian's motivation still applies with full force to **derived data**.

**Final delivery shape (hybrid):**
- **Blog prose + index:** build-time SSG → static HTML in the payload (all-crawler SEO + fast paint, satisfies `21`'s light-DOM-prose decision). Custodian is a **build-time dependency** (build fetches listed logs from the `/v1` read API), stays pure-JSON, no presentation, off the reader path.
- **Profile:** baked at build (already decided at charting).
- **Derived widgets (Steam/GitHub):** stay **client-runtime fetch** from custodian — decorative, not SEO/paint-critical; persona hides an empty widget (`12`/`14`). This is where instant freshness still matters and is preserved.

**Revises** the charting decision "blogs load at runtime"; **un-defers** `11`'s build-time blog snapshot (now the chosen mechanism, not a rejected one). **Unblocks `15`** with the constraint fixed: the persona framework must do **build-time static generation with a build-time data fetch from custodian**, plus client-side hydration for decorative widgets — it does **not** need ISR / on-demand-revalidation / SSR. **Feeds `16`** (blog delivery + SEO): SEO is general (all crawlers), delivered by prose-in-static-payload.
