# Spec: blank

Status: ready-for-agent

The Web Components UI library for the playground. This spec is the buildable
record for `blank`; it synthesises tickets `04` (design language), `05`
(Web Components research), `14` (component inventory), `21` (rendering
foundation), and `22` (accessibility), as revised by `15`/`16` (which removed
`<blank-markdown>` from v1). Ubiquitous language is per `CONTEXT.md`.

## Problem Statement

Mihir is building `persona`, the public website, and wants its look — a
recovered, deliberately monotone aesthetic — to be expressed as reusable UI
primitives rather than one-off page CSS. Two forces pull on that:

- **The aesthetic must be enforced, not re-implemented per page.** The
  deprecated site scattered its design language across CSS modules; the
  monotone ink ramp, single highlight yellow, lowercased chrome, and
  whitespace-driven hierarchy should live in one place so every surface reads
  as one system.
- **The library must not silently become `persona`-shaped.** If the UI layer
  quietly grows to know about "experiences," "logs," or "projects," it stops
  being reusable and becomes a private part of the website with extra
  ceremony. The chosen forcing function is publishing to public npm with
  semver from day one — a stranger being able to `npm install blank` is what
  keeps the API honest, even though no stranger is expected to.

So the problem is: produce the smallest set of genuinely reusable, publishable
UI primitives + design tokens that let `persona` (and any future project) be
built without re-deriving the aesthetic and without leaking website-specific
concepts into the shared layer.

## Solution

`blank` — a **primitives + tokens** Web Components library, TypeScript, built on
**Lit**, published to public npm as ESM with semver.

It ships two halves:

1. **A base / tokens stylesheet** — the monotone palette, type scale, dark
   theme, and global heading/link styling, exposed as CSS custom properties.
   Importing it restyles global `<a>` and headings, which is why there is no
   `<blank-link>` and no heading component.
2. **Five custom elements** — `<blank-stack>`, `<blank-card>`, `<blank-badge>`,
   `<blank-button>`, `<blank-stat>` — the deep, content-blind primitives that
   pass the **different-project test**: each would make sense in a project
   unrelated to a CV. Anything field-aware (a navbar, a hero, an experience
   card, a post-list item, the composed status *line*) is `persona`'s job and
   stays out of `blank`.

The governing principle is **`blank` grows with the implementations it
powers**: ship the minimal set, add on real need. Everything not in v1 is fog,
not a backlog.

## User Stories

Actors: **the consumer** (a developer wiring `blank` into a project — today
`persona`, tomorrow another project), **the maintainer** (Mihir, evolving the
published API), and **the reader** (an end user of a site built with `blank`).

### Tokens & aesthetic

1. As a consumer, I want to import a single base/tokens stylesheet, so that all
   global `<a>` and heading styling matches the aesthetic without me writing
   any CSS.
2. As a consumer, I want the design tokens exposed as documented CSS custom
   properties, so that I can reference the ink ramp, spacing scale, and type
   scale from my own page CSS.
3. As a consumer, I want headings to render centered, standing alone,
   lowercased, one weight, separated by whitespace (not a rule), so that I get
   the recovered hierarchy for free.
4. As a consumer, I want the type scale to be 20/16/15px (h1/h3/body) with
   hierarchy carried by spacing and a small size step — never weight — so that
   the "one weight everywhere" rule holds.
5. As a consumer, I want larger elements to carry *less* ink (h1 `#5F5F5F`, h3
   `#5C5C5C`, body full ink), so that presence is uniform and nothing
   dominates.
6. As a consumer, I want `#FBFF20` available strictly as a background highlight
   token, so that I cannot accidentally use it as text or an underline colour
   and break contrast/the aesthetic.
7. As a consumer, I want links to render as ink underlines (no yellow), so that
   emphasis stays monotone.
8. As a reader, I want a first-class dark theme, so that the site reads well in
   dark mode — which the monotone palette makes the better-looking default.
9. As a consumer, I want a single documented `600px` breakpoint and a
   `4·8·16·24·40·64·96px` spacing scale, so that layout decisions inherit the
   recovered rhythm.
10. As a consumer, I want long-form body prose to be Roboto Serif, justified,
    block-centered, at a ~64ch measure, so that reading surfaces match the
    aesthetic even though `blank` no longer renders markdown itself.
11. As a consumer, I want running body prose to keep its authored casing while
    all chrome and headings are lowercased, so that the one class of text meant
    to be read normally is visually marked.

### Components

12. As a consumer, I want a `<blank-stack>` that vertically flows its children
    with token spacing via a `space` attribute, so that I get consistent
    whitespace without hand-rolled margins.
13. As a consumer, I want `<blank-stack>` to be pure spacing with no
    heading/icon slots, so that a titled section stays a composition I own in
    `persona`, not a `blank` concern.
14. As a consumer, I want a `<blank-card>` — a padded box with a single default
    slot — so that I can wrap arbitrary content in the standard padded surface.
15. As a consumer, I want a `<blank-badge>` with text and an optional state dot
    (`active`/`idle` attribute), so that I can render a live-status surface as
    an instance of it rather than a bespoke component.
16. As a consumer, I want `<blank-badge>`'s dot to animate on state change but
    respect `prefers-reduced-motion`, so that the motion is a nice-to-have that
    never fights accessibility.
17. As a consumer, I want a `<blank-button>` with `primary`/`secondary`/`icon`
    variants, where `icon` takes an SVG via a slot, so that I get the standard
    action affordance without `blank` shipping an icon set.
18. As a consumer, I want `<blank-button>` to render a real native `<button>`,
    so that keyboard operability, Enter/Space activation, and the button role
    come for free.
19. As a consumer, I want `<blank-button icon>` to accept and pass an
    `aria-label` through to the real button, so that I can supply an accessible
    name for an icon-only action.
20. As a consumer, I want a `<blank-stat>` square stat tile (big mono number +
    label), so that I can render metrics in the validated viz primitive.

### Consuming, theming & packaging

21. As a consumer, I want ESM-only per-component subpath exports (`blank/button`
    …) plus a barrel, so that I pull in only what I use.
22. As a consumer, I want importing a component to auto-register its custom
    element (idiomatic side-effect entry), so that I can use the tag
    immediately without a `register()` call.
23. As a consumer, I want Lit to be a `peerDependency`, so that my dependency
    tree dedupes to one Lit copy and I never ship duplicate Lit majors.
24. As a consumer, I want theming to work purely through inherited CSS custom
    properties, so that one token channel styles both the shadow-DOM chrome and
    any light-DOM prose.
25. As a consumer, I want a committed, CI-drift-checked Custom Elements
    Manifest, so that I have machine-readable API docs and my tooling can
    introspect the components.
26. As a maintainer, I want every shipped component and exposed token to be an
    explicit, minimal API commitment, so that I can evolve `blank` under semver
    without accidentally exposing internals.
27. As a maintainer, I want internal class names and shadow structure to stay
    un-leakable, so that the public API is only what I deliberately export.

### Development & verification

28. As a maintainer, I want each component to have a plain static HTML demo page
    I can open in a browser with no build step, so that I can eyeball the look,
    the dark theme, and each variant/state the way I did with the `04`
    prototype.
29. As a maintainer, I want each component tested through its public
    custom-element interface in a real browser, so that my tests survive
    refactors of the internals.
30. As a maintainer, I want tests that assert on observable output, the
    accessibility tree, and computed styles — never on shadow-internal class
    names — so that the tests exercise behaviour, not implementation.
31. As a maintainer, I want the `22` accessibility floor (real `<button>`,
    unsuppressed native focus, reduced-motion, aria-label passthrough) covered
    by tests, so that the cheap-but-expensive-to-retrofit guarantees can't
    regress.

## Implementation Decisions

### Authoring & runtime

- **Lit, TypeScript.** `@customElement` decorator throughout. Lit is declared a
  **`peerDependency`** (~5 KB) so consumer trees dedupe to one copy. Chosen
  over vanilla `HTMLElement` deliberately (`21`) for idiomatic ergonomics,
  reactive properties, batched render, and CEM tooling — treating `blank` as a
  genuine Lit learning vehicle. The SSR penalty `05` attached to Lit is
  dissolved because `blank` never server-renders prose (see Encapsulation).

### Inventory (revised by 15/16)

- **One base/tokens stylesheet + five components.** `<blank-markdown>` is
  **removed from v1** — under `persona`'s SSG delivery (`15`/`16`) the
  markdown-to-HTML step dissolves into Astro's remark/rehype build, and its
  only consumer was the blog path SSG removed, so a runtime markdown component
  has no v1 consumer. This is a revision of `14`, which listed six components.
- **Consequence — uniform Shadow DOM.** `<blank-markdown>`'s light-DOM prose was
  `21`'s single encapsulation exception. With it gone, **all five shipped
  components use Shadow DOM**; the "mixed model" collapses to uniform shadow.
  The light-DOM-prose rationale is retained only as context should a runtime
  markdown component ever graduate from fog.
- The five components and their interfaces:
  - `<blank-stack>` — vertical flow, token-spaced via a `space` attribute. Pure
    spacing; **no** heading or icon slots.
  - `<blank-card>` — padded box, single default slot. No fields, no named slots.
  - `<blank-badge>` — text + optional state dot; dot is an attribute
    (`active`/`idle`), ink-only, animated on state change, animation gated on
    `prefers-reduced-motion`.
  - `<blank-button>` — action; `primary`/`secondary`/`icon` variants; renders a
    **real native `<button>`**; the `icon` variant takes an SVG via a slot and
    accepts + passes through an `aria-label`. No icon set shipped.
  - `<blank-stat>` — square stat tile: big mono number + label.

### Design tokens (from 04)

- **Type:** Roboto Mono 400 for all chrome (headings, UI, labels, code, axes);
  Roboto Serif for long-form body prose (same Roboto superfamily, shares the
  skeleton — serif may be nudged up a touch since mono reads optically larger).
  Poppins dropped. Type scale **20 / 16 / 15px** (h1 / h3 / body). Hierarchy
  from spacing + a small size step, **never weight** — one weight (400)
  throughout.
- **Ink & colour:** monotone ramp `#000000` / `#5F5F5F` + greys `#5C5C5C`
  `#969696` `#9C9A9A` `#9E9C9C` on `#FFFFFF`. Colour balanced against size for
  uniform presence: **h1 `#5F5F5F`, h3 `#5C5C5C`, body full ink.** `#FBFF20`
  kept **exact**, used **only as a background highlight** (`<mark>`) — no yellow
  text, no yellow underline. Links are **ink underline**; status dots are
  **ink**.
- **Dark theme** is first-class from the start, modelled as an independent token
  layer (ink → `#e9e9e9`, paper → `#0b0b0b`).
- **Headings:** centered, standing alone, lowercased (leading letter included),
  separated by whitespace only (~64px above / ~24px below an h3) — the old
  `h3:after` hairline rule is **dropped**.
- **Casing:** everything lowercased — chrome *and* headings — **except running
  body prose**, which keeps authored casing.
- **Body:** justified, its block centered on the page, measure ~64ch.
- **Spacing scale** `4·8·16·24·40·64·96px`; single `600px` breakpoint.

### Encapsulation & theming (from 21)

- **Shadow DOM for all five components** — real scoping; `adoptedStyleSheets`
  shares one constructed stylesheet across instances; internal class names stay
  un-leakable (decisive for a published lib).
- **Theming: CSS custom properties only, no `::part()` in v1.** Tokens are
  defaulted on `:root` in the base sheet and cross the shadow boundary by
  inheritance. This is *forced*, not chosen — custom properties are the only
  channel that both crosses the shadow boundary and (for any future light-DOM
  prose) styles unencapsulated content. `::part()` is withheld (each part would
  be permanent public API; `blank` is a fixed look, not a restyling kit) and can
  graduate on a real structural-override need. `::slotted()` is used internally
  as needed and is **not** a public commitment.

### Packaging (from 21)

- **ESM-only.** `"exports"` with per-component subpaths (`blank/button`, …) plus
  a barrel. **v1 ships the idiomatic auto-registering entry only** — importing a
  component defines its tag as a side effect; entry files are listed in
  `package.json` `sideEffects` so bundlers never prune registration. The
  bare-class / per-consumer-`register()` dual entry is deferred to fog
  (non-breaking additive).
- **Custom Elements Manifest** generated by `@custom-elements-manifest/analyzer`,
  committed, and CI-drift-checked (mirrors `13`'s treatment of the generated
  OpenAPI client). Referenced via the `"customElements"` field; doubles as the
  documentation source.
- Published to **public npm with semver from the start** — the forcing function
  that keeps the API from going `persona`-shaped.

### Accessibility floor (from 22)

`blank` bakes in only the near-free-but-expensive-to-retrofit shapes:

- `<blank-button>` is a real native `<button>` (real links are real `<a>`).
- The base stylesheet never suppresses native focus without a replacement.
- `prefers-reduced-motion` stops `<blank-badge>`'s animated dot.
- `<blank-button icon>` accepts + passes through an `aria-label` (mechanism
  only; the *value* is `persona`'s job).

Everything beyond this floor (WCAG audits, contrast tests) is out of scope.

### Development harness

- **Plain static HTML demo pages per component**, opened directly in a browser
  with no build step — the same shape as the `04` prototype
  (`prototypes/04-blank-design-language/index.html`). Each page renders the
  component's variants/states and can be flipped between light and dark. **No
  Storybook** in v1 (too heavy for five fixed components); a CEM-fed workbench
  is fog, graduating on component count or a second consumer.

### Toolchain

- The `@open-wc` ecosystem is the natural fit for the above (ESM, CEM analyzer,
  Web Test Runner). Exact build/lint config is left to implementation; the root
  `justfile` and pnpm-workspace posture come from `13`.

## Testing Decisions

- **What makes a good test here:** it drives a component the way a consumer
  does — mount the registered element in a real document, set attributes,
  properties, and slotted content, then assert on **observable output, the
  accessibility tree, and computed styles**. It never reaches into the shadow
  root to assert on internal class names, Lit internals, or DOM structure that
  isn't part of the contract. Lit, Shadow DOM, and `adoptedStyleSheets` are
  implementation detail.
- **The single seam:** each component's **public custom-element interface in a
  real browser**, via **Web Test Runner** (Playwright/Chromium launcher). A real
  browser is required — the seam must exercise Shadow DOM, `adoptedStyleSheets`,
  computed styles, and focus faithfully, which jsdom cannot. The base/tokens
  stylesheet is tested by asserting **computed styles on plain elements** after
  importing it (e.g. an `<a>` renders as an ink underline; an h1 computes to
  `#5F5F5F`; lowercasing applies to chrome/headings but not body prose).
- **Modules tested:** the base stylesheet and all five components. Behaviours
  worth pinning include: `<blank-stack>` `space` attr → computed gap;
  `<blank-badge>` `active`/`idle` → dot state + no animation under
  `prefers-reduced-motion`; `<blank-button>` renders a real `<button>`, is
  keyboard-operable, keeps a visible focus indicator, and forwards `aria-label`
  on the `icon` variant; `<blank-stat>` renders number + label; theming — a
  consumer overriding a documented custom property changes the rendered look
  across the shadow boundary.
- **Prior art:** the `04` design-language prototype is the reference for the
  static-HTML demo pages (per-variant, no build, light/dark). There is no other
  test prior art in the repo yet — `blank` is greenfield — so this spec
  establishes the seam that later TS-component tickets (e.g. `custodian`'s
  generated TS client tests under `13`) can look to for the "real environment,
  public interface only" stance.

## Out of Scope

- **`<blank-markdown>` / any runtime markdown renderer** — removed from v1 by
  `15`/`16`; markdown rendering lives in `persona`'s Astro remark/rehype build.
  A runtime markdown component graduates from fog only if a real second surface
  needs one.
- **Field-aware components** — navbar, hero, about, experience card,
  curated-projects/project card, skills, footer, post-list item,
  out-of-prose images, and the composed status *line*. All are `persona`
  compositions of the five primitives + base-styled headings.
- **Growth-fog primitives (`14`):** SVG line/area chart (waits on a dashboard
  project + a data-input shape), a bare standalone dot, an `inline`/cluster
  layout primitive, a first-party icon set (icons are a slot today), and a
  content-blind `<blank-section>`.
- **Bare-class / per-consumer `register()` dual entry (`21`)** — non-breaking
  additive; graduates when a second consumer needs rename-safe registration.
- **`::part()` theming surface** — withheld until a real structural-override need.
- **Storybook / a heavy component workbench** — static HTML demo pages instead;
  a CEM-fed workbench is fog.
- **Full accessibility (`22`):** WCAG 2.2 AA contrast audit + committed contrast
  tests, dark-theme + `#FBFF20`-as-text-background contrast verification,
  monotone focus-ring *design*, and *enforcing* (vs accepting) icon accessible
  names. Graduate only under a future dedicated a11y effort.
- **Visual-regression / screenshot testing** — the seam is the public DOM, not
  pixels; the demo pages cover eyeballing.
- **Adopting `blank` in other real projects** — it must be *publishable* and its
  API must not be `persona`-shaped, but wiring it into other projects is their
  own work (per the effort's out-of-scope ruling).

## Further Notes

- **Consumption contract with `persona` (`15`):** `persona` takes a
  `workspace:*` dependency on local `blank`, imports only its public `exports`,
  and registers components via the auto-registering `@customElement` entry in a
  client script (SSR posture B — `@lit-labs/ssr` stays absent). The base/tokens
  stylesheet is imported once globally. `persona`'s derived-data widgets are its
  own plain client-side custom elements, **not** `blank` components — they are
  field-aware and fail the different-project test.
- **Why publishing at all:** no stranger is expected to install `blank`;
  publishing to public npm with semver is purely the forcing function that
  stops the API drifting into a private `persona` shape (`CONTEXT.md`).
- **Aesthetic authority:** the `04` prototype
  (`prototypes/04-blank-design-language/index.html`) is the primary source for
  the tokens; this spec's token values are transcribed from its `## Answer`.
- **Cautionary lineage:** the whole "monotone, one spark, whitespace-driven"
  identity is *recovered* from the deprecated repo's `globals.css`, not
  invented — do not embellish it. Poppins and the `h3:after` hairline are
  confirmed dead and must not reappear.
