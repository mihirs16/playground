# blank: component inventory

Type: grilling
Status: open
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
