# blank: design language extraction

Type: prototype
Status: resolved

## Question

What is `blank`'s design language, stated explicitly as tokens and rules rather than implied by the old CSS?

Produce a concrete artifact to react to:

- **Token sheet** — type scale, the monotone ramp, spacing scale, the single highlight yellow, border/rule weights, breakpoints.
- **Two or three sample compositions** rendered as static HTML — a heading with body text, a blog excerpt, and a live-status line — so the feel can be judged rather than discussed abstractly.

Questions the prototype should force answers to:

- Is `#FBFF20` kept exactly, or tuned? It's an acid yellow that fails contrast badly as a text colour on white — is it strictly a background highlight, a rule/underline colour, or does it need a darker companion for text?
- Roboto Mono for *everything*, including long-form blog prose? Monospace body text at blog length is a real readability question the deprecated site never faced, because it had no blog. If a second face is needed, does that break the aesthetic?
- What does "lots of blank space" mean numerically — what's the spacing scale and the measure (max line length) for prose?
- Does the `h3:after` flex-rule detail generalise into a reusable primitive, or stay a one-off heading treatment?
- Is there a dark variant in the token structure from the start, even if unimplemented?

## Context

Takeable now — nothing blocks it. Use `/prototype`; keep it cheap and rough, and link the artifact from this ticket rather than pasting it in.

Harvested from the deprecated repo's CSS modules: Roboto Mono throughout, Poppins imported but barely used; `#000000`, `#5F5F5F`, `#FFFFFF`, greys `#5C5C5C` `#969696` `#9C9A9A` `#9E9C9C`; `#FBFF20` used exactly twice; `h3:after` as a 1px rule flexing to fill the line; a single `600px` mobile breakpoint.

The design language is being *recovered*, not invented — that's the constraint that makes this a prototype rather than a grilling.

## Prototype

[`prototypes/04-blank-design-language/index.html`](../prototypes/04-blank-design-language/index.html) — open in a browser, no build. Three resolutions of the recovered tokens, switchable via `?variant=A|B|C`, the floating bar, or ← →. Each renders the same token sheet + three sample compositions (heading/prose, blog excerpt, live-status line):

- **A — Purist**: Roboto Mono for everything incl. prose; `#FBFF20` kept exact, background-highlight + rule only; light only; `h3:after` stays a one-off.
- **B — Readable**: Newsreader for long prose, mono for headings/UI/code; darkened `#6b6d00` acid companion for text/links; `h3:after` generalised into a reusable `.rule-fill` primitive.
- **C — Dark-first**: dark token layer primary, acid the single accent carrying all emphasis; spacing scale surfaced.

## Answer

`blank`'s design language, recovered from the deprecated repo and confirmed against `styles/globals.css` (not just the harvested note), then resolved through the prototype:

**Type**
- **Roboto Mono, weight 400, everywhere for chrome** — headings, UI, code, labels, axes. The whole site's only `700` is one inline instance on the yellow-highlighted experience text.
- **Hierarchy: spacing does the majority of the work, with a small size step on top — never weight.** One weight (400) throughout; a gentle type scale of **20 / 16 / 15px** (h1 / h3 / body), well below the old CSS's 32/20/14. Headings are set apart mostly by whitespace (h3 gets ~64px above / 24px below).
- **Colour is balanced against size for uniform presence** — the larger an element, the more its ink fades, so no single element dominates: h1 (largest) sits at `#5F5F5F`, h3 at `#5C5C5C`, body at full ink. Consistent with the old CSS, which already faded `h2` to `#5F5F5F` while body stayed black.
- **Headings are centered and stand alone; body text is justified and its block is centered on the page** (centered heading over a centered, justified column). No hairline rules flanking headings.
- **Everything is lowercased, including the leading letter — except running body text.** All chrome (labels, meta, status lines, tile captions, axis ticks, affordances like "read more") *and* all headings (h1/h3) are lowercased. The **one exemption is running body prose** — a log post's body keeps its authored casing. So the casing itself marks the single class of text the reader is meant to read normally; everything framing it goes quiet in lowercase.
- **Roboto Serif for long-form prose** (the chosen companion, variant B). Stays inside the **Roboto superfamily** so it shares the mono's skeleton — the mono+serif mix reads well and just needs size fine-tuning at the design stage (mono looks optically larger than serif at equal px, so the serif will likely be nudged up a touch in the real tokens). Poppins is confirmed **dead** (imported, referenced nowhere) — dropped.

**Colour — fearlessly monotone, one spark**
- Monotone ink ramp: `#000000` / `#5F5F5F` + greys `#5C5C5C` `#969696` `#9C9A9A` `#9E9C9C` on `#FFFFFF`.
- `#FBFF20` kept **exactly** and used **only as a background highlight** (`<mark>`). No yellow text, no yellow underline. Links are **ink underline**; status dots are **ink**. In data viz the yellow is a single **spark** — a delta chip, a latest-point dot, or a featured tile's top edge — never a fill.

**Detail & structure**
- The recovered `h3:after` hairline rule is **dropped** — with centered, standing-alone headings the flanking rules were cut. (The site's one surviving structural line is the `<hr>`/dinkus concept if needed later, not a per-heading rule.)
- Spacing scale `4·8·16·24·40·64·96`px; prose measure ~64ch; single `600px` breakpoint (from the old site).

**Dark theme is first-class from the start** — validated in the prototype as an independent token layer (ink→`#e9e9e9`, paper→`#0b0b0b`), and looks *better* than light. The monotone palette makes this cheap.

**Data-viz primitives validated** (they belong in the component inventory): **square stat tiles** (boxed metric, big mono number, optional drawn SVG sparkline) and a **sleek line/area chart** mixing drawn SVG graphics with mono axis labels — graphic, not terminal-ASCII.

Prototype (the primary source): [`prototypes/04-blank-design-language/index.html`](../prototypes/04-blank-design-language/index.html).

## Blocks

`14` blank component inventory
