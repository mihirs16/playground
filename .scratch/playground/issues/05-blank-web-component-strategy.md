# blank: Web Components strategy

Type: research
Status: resolved

## Question

Lit or vanilla custom elements, and what does the styling and distribution strategy have to be for a publicly published Web Components library?

- **Lit vs vanilla** — what does Lit actually buy at this scale (reactive properties, templating, ~5KB), and what does it cost (a runtime dependency in every consumer, a version to keep current)? For a minimal monotone library, is vanilla `HTMLElement` genuinely sufficient?
- **Shadow DOM or light DOM?** This is the crux. Shadow DOM gives encapsulation but makes consumer theming hard and complicates SSR. Light DOM with a prefixed class convention is themeable but leaks. What's the current state of practice?
- **Constructable stylesheets / `adoptedStyleSheets`** — browser support and whether they solve the per-instance style duplication problem.
- **SSR and hydration** — does Declarative Shadow DOM work well enough for a static site to serve pre-rendered component markup? What's Lit SSR's maturity, and which frameworks support it? This directly gates `15`.
- **Theming API** — CSS custom properties crossing the shadow boundary, `::part()` and `::slotted()`. What does a consumer need to restyle `blank` for a different project, given it's meant to be reused?
- **Packaging** — how a Web Components library ships to npm: ESM only, individual entry points vs a bundle, `customElements.define` side effects and tree-shaking, TypeScript declarations, and the `custom-elements.json` manifest for editor support.
- **Framework interop** — the current state of React's custom-element support (notably improved in React 19), plus Astro, Svelte, and Vue, so the library doesn't foreclose consumers.

## Context

AFK — resolve with a `/research` subagent against primary sources (Lit docs, WHATWG/MDN, framework docs, `custom-elements-manifest`). Facts, not decisions; the Lit-vs-vanilla call itself may need a follow-up grilling once the facts land.

Already decided: `blank` is TypeScript (Web Components foreclose the choice) and publishes to public npm with semver from the start. That raises the stakes on the packaging and theming questions — a published API is one you have to live with.

## Blocks

`15` persona framework

## Answer

Facts gathered against primary sources (Lit, WHATWG HTML spec, MDN, Node, webpack, React/Astro/Svelte/Vue docs, Custom Elements Manifest). Full findings with per-claim citations: [`research/05-blank-web-components-strategy.md`](../research/05-blank-web-components-strategy.md).

- **Lit vs vanilla** — Lit is a **~5 KB min+gzip** runtime dependency (Lit's own homepage figure) that gives property↔attribute sync, microtask-batched re-render, and `html` expression-level DOM diffing for free. Vanilla `HTMLElement` forces you to hand-write all three (`observedAttributes`/`attributeChangedCallback` + getters/setters, your own batching, your own diffing — naive `innerHTML=` re-parses and destroys subtree state). Costs of Lit: a versioned runtime every consumer loads, plus a duplicate-Lit-versions bloat risk in a consumer's tree. For a small monotone set, vanilla is genuinely tractable; the trade is hand-rolled-and-maintained vs 5 KB-off-the-shelf.
- **Shadow vs light DOM** — Shadow DOM gives real bidirectional DOM + style scoping (MDN); Lit's own docs call light-DOM rendering **"generally not recommended"** (loses scoping, breaks composition, leaks internal class names as public API). No WHATWG/W3C stance either way — it's an author call, and the dominant first-party tooling defaults to Shadow DOM.
- **`adoptedStyleSheets`** — **Baseline widely available since March 2023**; one `CSSStyleSheet` shared by reference across many shadow roots solves per-instance style duplication. This is the correct mechanism (and what Lit uses under the hood).
- **SSR / DSD (gates `15`)** — **Declarative Shadow DOM** (`<template shadowrootmode>`) is a *platform* feature, **Baseline since Aug 2024** (Chrome/Edge 111+, FF 123+, Safari 16.4+) — a static site can serve pre-rendered shadow markup with **no JS**. But **Lit's own SSR (`@lit-labs/ssr`) is explicitly Labs/experimental** with named limitations. So: **vanilla → DSD works everywhere, library-independent; Lit → pre-rendering `blank` relies on non-stable tooling.** Astro and Eleventy (WebC) both have first-party patterns for plain custom elements without a stable Lit-SSR story.
- **Theming** — CSS custom properties **inherit through** the shadow boundary automatically (the low-friction channel); `::part()` and `::slotted()` are opt-in and require the author to annotate `part="…"` / write `::slotted()` rules per element. Theming is a *designed, published surface*, not free.
- **Packaging** — ESM-only is valid but narrower than Node's documented default; per-component entry points work via `exports` subpaths; **`customElements.define`-on-import is a genuine side effect that fights tree-shaking** unless reconciled via the `sideEffects` field (or by not auto-registering and exposing a `register()`/class instead); ship `.d.ts`; **Custom Elements Manifest v2.1.0** (generated from source) is the standard path to IDE support.
- **Framework interop** — **no mainstream consumer is foreclosed.** React 19 closed the historic gap (100% on Custom Elements Everywhere, documented props-vs-attributes + `on<Event>` strategy); Astro, Svelte, Vue all have mature, documented interop.

**Decision surfaced, not made** (as this ticket's Context anticipated): the Lit-vs-vanilla + Shadow-vs-light-DOM call is now a sharp grilling question — graduated to a new ticket. The pivotal fact for it is the SSR asymmetry above.

_Note: two orphaned Opus subagents (from a superseded first attempt) independently corroborated the Lit-vs-vanilla, `adoptedStyleSheets`, Shadow-vs-light-DOM, and theming findings against the same primary sources._
