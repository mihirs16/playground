# persona: framework

Type: grilling
Status: resolved
Blocked by: 23

## Question

What framework does persona use?

Judge against persona's actual constraints, which are now unusually specific: it consumes a Web Components library, it bakes profile content at build time, and it loads blogs plus live status at runtime in the browser. **NB: `23` revisits the "blogs load at runtime" assumption** — the viable framework set depends on its outcome (pure SSG vs ISR/on-demand-revalidation vs SSR), which is why this ticket now blocks on `23` rather than `21` directly (`23` itself depends on `21`).

- **Astro** — islands architecture, first-class Web Components support, excellent static output, markdown-native. The strongest fit on paper for a content site consuming custom elements.
- **Next.js** — familiar from the deprecated site, but the App Router is a lot of machinery for a static site, and React's custom-element interop was historically poor (materially improved in React 19 — `05` should confirm the current state).
- **SvelteKit** — good Web Components interop, small output, adds a language to learn.
- **11ty or plain Vite** — minimal, no framework opinions, most manual work. Genuinely viable given how little persona does: it's a static shell that fetches.

Also settle:

- **Does persona need a framework at all?** If profile is baked and everything else is client-fetched, this may be closer to a handful of HTML files plus a build step than an app.
- **Client-side routing** — blogs load at runtime, so do blog permalinks work as real URLs? This is the crux of `16`.
- **How profile gets baked** — build-time fetch from custodian, meaning a rebuild on profile change, which was accepted as fine for yearly edits.

## Context

Blocked on `05`, because whether Declarative Shadow DOM and Lit SSR work in a given framework directly determines which frameworks are viable. Choosing the framework before knowing that risks picking one that can't pre-render `blank` at all.

Standing constraint: polyglot is incidental. Do not pick a framework for novelty — pick the one that renders Web Components and markdown with the least ceremony.

## Answer

**persona is Astro (`output: 'static'`), a pure SSG with no render tier** — the lowest-ceremony fit for `23`'s fixed shape (build-time bake, prose-in-payload, client-hydrating `blank` chrome, client-fetch derived widgets). Astro was chosen on the merits, not novelty: it *reduces* bespoke surface (markdown pipeline, blog routing, index generation come for free) rather than adding a language, so "polyglot is incidental" doesn't bite. The minimal Vite/11ty case was considered and rejected — it trades a dependency you'd learn anyway for a pile of glue you'd own forever. **Does persona need a framework?** Yes — Astro *is* the "handful of files plus a build step," just with the content-site plumbing already built.

**Routing — fully static, no client-side router.** Each log is a real prerendered file via `getStaticPaths()` over custodian's slugs (`09` — slug is a log's sole identity, maps 1:1 to the output path); every page is a physical file CloudFront serves directly, deep permalinks survive a hard refresh, no SPA router, no client navigation. This dissolves the "do runtime-loaded blogs have real URLs?" worry the ticket shared with `16` — `23` made blogs build-time, so the question is moot. **`build.format: 'directory'`** gives clean `/logs/<slug>/` URLs (never `.html`/`index.html` in the address bar). Because `19` fronts a **private S3 bucket via CloudFront+OAC** (not a public S3 website endpoint), the raw-S3 origin won't auto-resolve subfolder indexes — so a **CloudFront Function (viewer-request) rewrites `/logs/<slug>/` → `/logs/<slug>/index.html`**, ~15 lines, owned by **`deed`**. This is the current mainstream AWS pattern (OAC + CloudFront Functions, the canonical index-rewrite example) and deliberately avoids the public-website-endpoint shortcut that would let clients bypass CloudFront (undercutting `10`'s edge WAF/rate-limiting and `19`'s perimeter). *Adds a line item to `deed` and touches the CloudFront↔origin fog.*

**Profile stays in custodian, baked at build** — but the *rationale shifted*. `03`'s original reason (escape rebuild-to-edit) is **dead**: `23` reintroduced rebuild-to-edit for all baked content, so custodian storage buys profile zero freshness now. What keeps it there is a **new** reason the grilling surfaced: **a future QnA bot/assistant answering from the profile is a retrieval consumer** — the same "systems search/retrieve/reference" need cited for blogs — and a bot reads an API/queryable store, not persona's build artifacts. So `03`/`08`/`09`/`17` are **not** revised (moving profile to repo files was considered and rejected on this basis). persona bakes profile via a build-time `fetch()` against custodian's **public `/v1`** read surface (no admin token in the build; a custodian-down build fails loudly, which is correct); the bot later hits that same public surface. The QnA bot itself is out of this map's scope — it's the standing reason profile stays API-backed, not a deliverable here.

**Monorepo wiring — `workspace:*`, imports through public exports only.** persona depends on the local `blank` package via pnpm's `workspace:*` (`13`), so dev/build are always on head — which also lets logs *about* `blank` import and render live `blank` components. The **forcing-function invariant** (charting: publishing keeps `blank`'s API from going persona-shaped) is preserved by importing **only through `blank`'s published `exports` map** (`21`'s per-component subpaths + barrel), never a deep `../blank/src/...` reach. **Registration:** persona loads `blank`'s auto-registering `@customElement` entry (`21`) in a **client-side script**; Astro emits inert `<blank-*>` tags at build and the browser upgrades/hydrates them — matching `21`'s posture B, so `@lit-labs/ssr` stays absent.

**Base stylesheet imported once, globally** in Astro's root layout — `14`'s base/tokens stylesheet styles the light-DOM blog prose *and* supplies the custom-property tokens that cross into `blank`'s shadow chrome (`21`'s only cross-boundary channel). **Derived widgets are plain client-side custom elements, not Astro islands** — because `blank` components are raw custom elements (not framework components), they hydrate via native custom-element upgrade, not Astro's `client:*` directives; each widget fetches custodian on `connectedCallback` and hides itself on empty (`12`).

**Embedding `blank` in logs is supported nearly for free** (feeds `16`): raw HTML is part of GFM core (`03`), so `<blank-stat>…` in a log is allowed grammar, not novel syntax; the only knob is letting raw HTML through the build pipeline (`rehype-raw`/`allowDangerousHtml`), safe because logs are single-trusted-author. Global `blank` registration then upgrades any `<blank-*>` tag the markdown emits.

**Surfaced for `16` (does not resolve it here):** under SSG + prose-in-payload, the build-time markdown renderer **must** be Astro's own remark/rehype pipeline — it *cannot* be `<blank-markdown>`, because rendering a Lit component at build = `@lit-labs/ssr`, which `21` banned. So `<blank-markdown>` is **not on the blog critical path**, and its `14` raison d'être (centralise rendering) evaporates under SSG. **Owner's steer: `<blank-markdown>` should leave `blank`.** `16` must bank one of: **(a)** it becomes a *persona*-owned Web Component for runtime-markdown contexts, or **(b)** it dissolves entirely into persona's Astro build (no markdown *component* in v1, since SSG leaves no runtime markdown consumer) — leaning (b). Either way **`16` revises `14`**.
