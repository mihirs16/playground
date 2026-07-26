# blank: rendering foundation (Lit vs vanilla, Shadow vs light DOM)

Type: grilling
Status: open
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

`15` persona framework — its viable framework set depends on whether `blank` renders via universal DSD (vanilla) or experimental Lit SSR (Lit).
