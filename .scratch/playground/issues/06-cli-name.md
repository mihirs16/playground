# CLI: name

Type: grilling
Status: resolved

## Question

What is the CLI called?

You asked for a one-word suggestion. Shortlist, with the reasoning:

- **`recess`** — *recommended.* Playground vocabulary, exactly like `blank` and `persona` are drawn from their own domains. Six characters, easy to type, no meaningful npm/Homebrew collision, and it reads naturally as a command: `recess publish`, `recess new post`.
- **`scribe`** — names the function rather than the setting: authoring content. Six characters. Slightly more literal, slightly less playful.
- **`quill`** — shortest at five, authoring-flavoured, but sits oddly beside the playground metaphor.
- **`warden`** — pairs semantically with `custodian`, but two guardian-words for two components is confusing rather than cohesive.

Rejected: `chalk` (perfect playground imagery, but collides with one of the most-downloaded npm packages in existence), `curator` (too close to `custodian`), `groundskeeper` (far too long to type repeatedly).

## Context

Takeable now — nothing blocks it, and it's the smallest ticket on the map. Worth resolving early because the name appears in every later cli ticket, the monorepo layout, and the spec documents.

Naming is a `/domain-modeling` act: record the chosen name and its rationale in `CONTEXT.md` alongside `custodian`, `blank`, and `persona`, so the four names are legible as a set.

Check the chosen name against npm, Homebrew, and Go module paths before committing, since distribution is still open in `17`.

## Blocks

`17` cli language, UX and distribution

## Answer

**The CLI is `broom`.**

Chosen from an expanded shortlist after the framing was sharpened: the cli is *a tool used to configure, edit and write*, which pointed at the instrument rather than the playground setting.

Why it works:

- **It pairs with `custodian`.** A broom is the custodian's own implement — the tool the keeper actually holds. The two names read as a set rather than as two independent choices, which is exactly the relationship the components have: `broom` exists only to drive custodian's API.
- **Humble, which is honest.** The cli is not clever machinery; it moves content from your editor into custodian. A grand name would oversell it.
- **Five characters, no awkward keys**, and it reads naturally in every command position: `broom publish`, `broom new post`, `broom media add`.
- **It shadows no standard Unix command**, so there's no PATH hazard.

### Namespace check

| Registry | Result |
| --- | --- |
| npm, bare `broom` | **Taken** — `broom@0.1.2`, "Application level flow-control library". Abandoned; the author's homepage uses `twitter.com/#!/` hashbang URLs, dating it to roughly 2012. |
| npm, scoped | Free — both `@broom/cli` and `@mihirs16/broom` are unregistered. |
| Homebrew | Free — no formula of that name. |
| Go modules | Non-issue — module paths are repo-namespaced, e.g. `github.com/mihirs16/broom`. |
| crates.io | **Not verified.** The crates.io API refused the request under its data-access policy. Check by hand if the cli turns out to be Rust. |

The dead npm package only constrains things if the cli ships via npm, which `17` hasn't decided. If it does, use a scope. Homebrew and GitHub Releases are unaffected.

### Alternates considered and rejected

`etch` (4 chars, and the recommendation at the time — "to inscribe", with an Etch-a-Sketch echo back to the playground), `nib` (3 chars, the pen's business end), `vellum` (the writing surface, pairing with `blank`), `folio`, `chisel`, `tend`, `lathe`, `porter`.

Ruled out on collisions rather than taste: `quill` (the rich-text editor), `ink` (React for CLIs), `stencil` (a Web Components compiler — the worst possible clash given `blank`), `graphite` (the `gt` CLI), `codex`, `chalk`, `type` (a POSIX shell builtin), `draft` (collides with the domain's own draft flag), and `scratch` (collides with this repo's `.scratch/` convention).

### Follow-through

The name is recorded in `CONTEXT.md` alongside `custodian`, `blank`, and `persona`, so the four read as a deliberate set.
