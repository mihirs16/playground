# persona: framework

Type: grilling
Status: open
Blocked by: 21

## Question

What framework does persona use?

Judge against persona's actual constraints, which are now unusually specific: it consumes a Web Components library, it bakes profile content at build time, and it loads blogs plus live status at runtime in the browser.

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
