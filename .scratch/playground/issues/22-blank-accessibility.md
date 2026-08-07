# blank: accessibility

Type: grilling
Status: resolved
Blocked by: 14

## Question

What are the accessibility commitments of `blank`'s six v1 components + base layer, and where does that responsibility split between `blank` and persona?

- **Contrast** — the monotone palette makes contrast unusually tractable, but it must be *verified* against WCAG in both light and dark themes: full-ink body vs the faded-by-size heading inks (`04` deliberately fades larger elements — h1 `#5F5F5F` — which is exactly where contrast gets risky), and the `#FBFF20` highlight as a text background.
- **Focus & keyboard** — `<blank-button>` (incl. the `icon` variant with no visible text) and any interactive badge/link states need visible focus rings and keyboard operability in a monotone world where colour can't signal focus.
- **Semantics across the shadow boundary** — `<blank-button>` must expose a real button role/label; the `icon` variant needs an accessible name (aria-label contract with persona, since persona slots the SVG). `<blank-badge>`'s animated state dot needs a non-visual state announcement, and its animation must respect `prefers-reduced-motion`.
- **`<blank-markdown>`** — rendered prose semantics (heading order, alt text passthrough, code block labelling) and whether the client-side render is announced to AT.
- **The split** — which guarantees are `blank`'s (baked into the component) vs persona's responsibility (correct heading order on a page, meaningful alt text, aria-labels for slotted icons)?

## Context

Graduated from the map's "blank: accessibility" fog once `14` fixed the component inventory. Dark mode is not open here — `04` settled it as a first-class token layer; this ticket is contrast/focus/keyboard/semantics of the actual `14` components. Coordinate with `04` (token inks) and `14` (the inventory + the `blank`-vs-persona line, which this extends into a11y responsibility).

## Answer

**Accessibility is a cheap floor for v1, not a commitment.** Full a11y (WCAG conformance, audits, tests) is explicitly *not* a priority for this destination. `blank` bakes in only the handful of things that cost nothing to state now **and** are expensive to retrofit if the components bake in the wrong shape; everything else is fog.

**`blank` guarantees (baked in, near-free):**

1. **`<blank-button>` renders a real native `<button>`** (not a styled `<div>`) — keyboard operability, Enter/Space activation, and button role for free. Links are real `<a>`.
2. **Native focus is never suppressed** — the base stylesheet won't `outline: none` without a replacement. No custom monotone focus-ring *design* is promised (fog); the guarantee is only "don't break the default."
3. **`prefers-reduced-motion` is respected** by `<blank-badge>`'s animated state dot — it stops animating under reduced-motion.
4. **`<blank-button icon>` accepts and passes through an `aria-label`** to the real button — the *mechanism* exists; supplying a meaningful value is persona's job.

**The split.** `blank` owns: native semantics, focus-not-broken, reduced-motion, and the aria-label *passthrough mechanism*. **persona owns**: correct heading order per page, meaningful alt text, and the actual aria-label *values* for slotted icons.

**Explicitly fog (not v1):** WCAG 2.2 AA contrast audit + committed contrast tests; dark-theme pairing verification; `#FBFF20`-as-text-background contrast; monotone focus-ring design; `<blank-markdown>` AT announcement / render semantics; and *enforcing* (vs merely accepting) icon accessible names. These graduate if a real a11y effort is ever taken up — a fresh effort, not this destination.
