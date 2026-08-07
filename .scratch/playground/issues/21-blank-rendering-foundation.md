# blank: rendering foundation (Lit vs vanilla, Shadow vs light DOM)

Type: grilling
Status: resolved
Blocked by: 05

## Question

Now that the facts are in (ticket `05`), make the two coupled foundational calls for `blank`, plus the commitments that follow from them:

- **Lit vs vanilla custom elements.** Is a ~5 KB versioned runtime dependency in every consumer worth property/attribute sync + batched re-render + template diffing off the shelf, or does a minimal monotone set justify hand-writing those against `HTMLElement` to ship zero runtime? Weigh the duplicate-Lit-versions risk for a *published* library against the maintenance cost of a hand-rolled base class.
- **Shadow DOM vs light DOM.** Shadow DOM is the first-party default (real scoping, but theming/SSR are designed surfaces); light DOM leaks internal class names as public API. For a library that must stay reusable and not become persona-shaped, which encapsulation model?
- **The pivotal coupling — SSR.** `05` found the asymmetry: vanilla → Declarative Shadow DOM is a platform feature that works everywhere; Lit → pre-rendering `blank` depends on `@lit-labs/ssr`, which Lit itself labels experimental. Does `persona` actually need `blank` server-pre-rendered, or is client-side island hydration acceptable? This answer gates how much the Lit choice costs, and it should be settled here **before** `15` picks persona's framework.
- **Downstream commitments to lock while deciding:** the theming API surface (which CSS custom properties `blank` exposes on `:host`, which elements get `part=`); the packaging stance (ESM-only? auto-register-on-import vs explicit `register()`, and the `sideEffects` reconciliation); and committing to a generated Custom Elements Manifest.

## Context

Graduated from `05` (research), whose Context anticipated this follow-up grilling. Read [`research/05-blank-web-components-strategy.md`](../research/05-blank-web-components-strategy.md) first — it holds the cited facts each option turns on.

Standing constraints: `blank` is TypeScript, publishes to public npm with semver from the start (every choice here is an API commitment you live with), and its interface must not become persona-shaped. Polyglot is incidental — don't pick Lit for novelty or vanilla for purism; pick on the SSR/size/maintenance trade.

## Blocks

`15` persona framework — its viable framework set depends on whether `blank` renders via universal DSD (vanilla) or experimental Lit SSR (Lit). *(Superseded mid-resolution: `15` now blocks on `23`, the blog-delivery-model ticket surfaced here, which itself depends on this ticket.)*

## Answer

The two coupled foundational calls, plus the downstream commitments, all resolved by grilling against `05`'s facts and one supporting research pass on how real OSS projects ship Web-Component blogs.

**1. SSR posture — (B) content-in-payload, chrome hydrates.** persona ships blog *prose* as plain semantic HTML in the initial server response (crawlable, first-paint before JS); decorative `blank` chrome hydrates client-side as islands. **Time-to-paint is an explicitly watched metric** — above-the-fold chrome (navbar/hero) is where hydration flash could show, not the blog body. Rejected: (A) full DSD pre-render (buys crawlable stat tiles nobody searches for, at the cost of DSD-emission tooling); (C) client-only prose (gambles blog SEO on JS execution).

**2. Authoring layer — Lit** (~5 KB), as a **`peerDependency`** so a consumer tree dedupes to one copy (kills `05`'s duplicate-Lit-majors risk). `@customElement` decorator throughout. This was a deliberate override of the size/paint-minimising vanilla case: chosen for idiomatic ergonomics, reactive properties, batched render, and first-class CEM tooling, treating `blank` as a genuine Lit learning vehicle. The SSR penalty `05` attached to Lit is dissolved by decision (1)+(3) — nothing renders shadow prose on the server, so `@lit-labs/ssr` is never a dependency.

**3. Encapsulation — mixed model.** Shadow DOM for the five chrome components (stack/card/badge/button/stat) — real scoping, `adoptedStyleSheets` shares one sheet across instances, and (decisive for a *published* lib) internal class names stay un-leakable so the public API is only what's deliberately exposed. **Light DOM for `<blank-markdown>`'s prose** — the one blessed exception, because the prose must land crawlable + fast-painting in the payload (decision 1) and be styled by the global base/tokens sheet (Roboto Serif, justified). This is not inconsistency — the encapsulation model follows the SEO/paint contract. Verified by research: the ecosystem consensus (WebC, Enhance, **and lit.dev's own site**) is prose-in-light-DOM, chrome-in-shadow; **even Lit's team does not use `@lit-labs/ssr` for lit.dev's prose**, and that package remains undated Labs software. All-light-DOM was weighed and rejected: it surrenders class-name-leak protection on a published library *and* makes Lit the non-idiomatic tool (light-DOM Lit is "generally not recommended"), so it would have reopened the Lit choice. Uniform-shadow+DSD was weighed and rejected once research showed `@lit-labs/ssr`-for-prose is exactly what the ecosystem avoids.

**4. Theming surface — CSS custom properties only, no `::part()` in v1.** The monotone palette, 20/16/15 scale, and spacing are exposed as documented custom properties defaulted on `:root` in the base sheet. **Tokens-as-custom-properties is forced, not chosen** — it is the only channel that both crosses the shadow boundary (by inheritance) into the five chrome components *and* styles the light-DOM prose. `::part()` is withheld (every exposed part is permanent public API, and `blank` is a *fixed* monotone look, not a restyling kit; parts can graduate on a real structural-override need). `::slotted()` is used internally as needed — not a public commitment.

**5. Packaging — ESM-only.** `"exports"` with per-component subpaths (`blank/button`, …) plus a barrel, so a consumer pulls only what it uses. **Ship the idiomatic auto-registering entry only for v1** (`@customElement` side-effect defines the tag on import; the entry files are listed in `package.json` `sideEffects` so bundlers never prune the registration). Lit is a `peerDependency`. The bare-class / per-consumer-`register()` dual entry (the Shoelace/Material-Web pattern) is deferred to fog — it is a **non-breaking additive** entry shape, added when a second consumer actually needs rename-safe registration.

**6. Custom Elements Manifest — yes.** Generated from source by `@custom-elements-manifest/analyzer`, committed, CI-drift-checked (mirrors `13`'s treatment of the OpenAPI-generated client). Referenced via the `"customElements"` field. Near-free with Lit's decorators/JSDoc, and the honest artifact for a published library whose point is being consumable elsewhere; doubles as the documentation source.

**Surfaced and ticketed:** resolving (1) exposed that the charting decision "blogs load at *runtime* from custodian" contradicts prose-in-the-payload. Created `23` (persona: blog delivery model — runtime fetch vs build-time SSG/ISR), blocked by this ticket, now blocking `15` (rewired off `21`). Invariant held: blogs stay in custodian's SQLite for indexing/source-of-truth; `23` is about *delivery* only.

**Supporting research:** one Sonnet subagent pass (not persisted as its own doc) confirmed the ecosystem pattern and `@lit-labs/ssr`'s Labs status; key sources — [lit/lit.dev](https://github.com/lit/lit.dev), [Eleventy+Lit](https://lit.dev/blog/2022-02-07-eleventy/), [lit#3353 SSR status](https://github.com/lit/lit/discussions/3353), [Enhance: Shadow DOM Not by Default](https://enhance.dev/blog/posts/2023-08-18-shadow-dom-not-by-default).
