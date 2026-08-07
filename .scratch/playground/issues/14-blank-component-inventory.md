# blank: component inventory

Type: grilling
Status: resolved
Blocked by: 04

## Question

Which components does `blank` actually ship, and where is the line between a `blank` primitive and a persona-specific composition?

- **Inventory** — the deprecated site implies navbar, hero, about, experience, projects, skills, footer, and button. Which of those are genuinely reusable primitives versus page sections that belong in persona? A `<blank-experience-card>` is almost certainly persona's business; a `<blank-button>` is not.
- **New for blogs** — what does long-form markdown rendering need? Prose container with a measure, headings, code blocks with syntax highlighting, blockquotes, inline highlight-yellow marks, images with captions, a post-list item.
- **New for live status** — a status line component, including its idle and stale states (coordinate with `12`, which decides what those states even are).
- **Layout primitives** — given "lots of blank space" is a defining property, does `blank` ship spacing/stack/rule primitives rather than leaving whitespace to consumers? This is arguably the library's whole point.
- **Syntax highlighting** — a monotone design with one accent is hostile to conventional multi-colour code themes. Does code get a monotone treatment with yellow accents, and is highlighting done at build time in custodian or at runtime in the browser? Real weight cost either way.
- **The API-surface line** — what's the minimum inventory that makes `blank` genuinely usable in a *different* project? That's the test that keeps it from becoming persona-shaped, which was the whole reason for publishing it.

## Context

Blocked on `04`, which produces the tokens and sample compositions this inventory is drawn from.

Already decided: public npm with semver from the start. Every component named here is an API commitment, so the bias should be toward shipping fewer, deeper components rather than a wide shallow set. `/codebase-design` is the relevant vocabulary if the interface-depth question gets contentious.

## Answer

**Governing principle — `blank` is a primitives + tokens library, not a component kit.** A thing belongs in `blank` iff it passes the **different-project test**: would it make sense in a project unrelated to the CV? Anything *field-aware* (knows an "experience," a "log's" fields, a "project") fails and lives in persona. Corollary that resolves most "should we ship X?" by default: **`blank` grows with the implementations it powers** — ship the minimal set, add on real need; unbuilt = fog, not v1. This is what keeps the public API from going persona-shaped, which was the whole reason to publish it.

### v1 deliverables

**Non-component deliverable — the base / tokens stylesheet** (opt-in import). The "tokens" half of the identity: monotone palette, 20/16/15px type scale, dark theme (first-class per `04`), global `<a>` ink-underline, and heading styling — **centered, standing alone, lowercased, one weight (400), separated by whitespace not a rule** (~64px above / ~24px below an h3; the `h3:after` hairline is dropped per `04`). Importing it restyles global `<a>` and headings, so no `<blank-link>` component is needed.

**Components (6):**

| Component | Role | Notes |
|---|---|---|
| `<blank-stack>` | vertical flow, token-spaced (`space` attr) | pure spacing; **no** heading/icon slots — a titled section is a persona composition of a stack + a base-styled heading |
| `<blank-card>` | padded box, single default slot | just padding/spacing; no fields, no named slots |
| `<blank-markdown>` | markdown → rendered page content | client-side renderer + monotone runtime highlighting (weight/opacity/italics, **no long snippets** — long code redirects to a code host); the one reading surface; centralises all rendering so custodian stays a raw-markdown store that never learns rendering |
| `<blank-badge>` | text + optional state dot | dot is an **attribute** (`active`/`idle`), **animated** on state change, ink-only; the "live status" surface is an instance of this, not its own component |
| `<blank-button>` | action | `primary` / `secondary` / `icon` variants; icon is a **slot** (no icon set shipped) |
| `<blank-stat>` | square stat tile | number + label |

### Persona, not `blank`

navbar, hero, about, experience card, curated-projects/project card, skills, footer, post-list item, out-of-prose images (cover/embeds), and the status *line* (label + dot). All are compositions of the six primitives above + base-styled headings — each knows its own *fields*, and none of that knowledge leaks into `blank`.

### Fog — grows on real need

- **SVG line/area chart** (the `04`-validated viz primitive) — deferred; graduates when a dashboard project actually uses `blank`. Data-shape questions (input format, axes, scales) aren't answerable until a real surface asks.
- **Bare standalone dot** — only if something needs a dot outside a badge.
- **`inline` / cluster layout primitive** — added if a repeated horizontal-spacing need proves out (dropped from v1: raw flexbox covers nav/tags today).
- **Shipped icon set** — icons are a slot today; a first-party set graduates only if wanted.
- **`<blank-section>`** — a content-blind titled/icon section primitive, *only if* every persona page turns out to repeat identical section furniture; even then not by overloading `stack`.

### Cross-ticket flags

- **→ `21` (rendering foundation):** `<blank-markdown>` now (a) ships a GFM parser + monotone highlighter as **runtime weight** — real input to the Lit-vs-vanilla budget — and (b) deep-styles rendered markup, where Shadow DOM's `::slotted()` reaches only the top-level node. How prose subtree styling is achieved (light DOM vs renderer output inside the shadow tree) is `21`'s call; the one-component inventory holds either way.
- **→ `16` (persona blog delivery & SEO):** rendering is client-side, so blog HTML isn't in the initial payload unless `<blank-markdown>` is pre-rendered (Declarative Shadow DOM / SSR, per `05`). `<blank-markdown>` must be SSR-able or `16` has a crawlability problem.
- **Map cleanup (not this ticket's decision):** the `04` Decisions-so-far line contains a contradictory clause ("`h3:after` kept verbatim") against its own authoritative `## Answer` (rule **dropped**). Cleaned up in the same map edit.
