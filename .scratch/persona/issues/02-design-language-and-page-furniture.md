# 02 — Design agreement: look & feel + `blank` page furniture

**What to build:** The shared visual system and the composed page furniture every
surface page is built from, agreed and demoable before the content surfaces land.
`persona` applies one monotone, whitespace-heavy identity with a first-class dark
theme, sets long-form prose in the reading serif with authored casing preserved
while chrome and headings are lowercased mono, and composes its own page
furniture from `blank`'s primitives: navbar, hero, about block, experience card,
curated-project card, skills, footer, post-list item, and the status *line*
(label + dot). Each furniture piece knows its own *fields*; none of that
field-awareness leaks back into `blank` (assume `blank`'s primitives already
exist and are consumed through its published `exports`). The tokens that cross
into `blank`'s shadow chrome flow only through the shared base/tokens stylesheet.
A kitchen-sink / styleguide page renders the full furniture set in both themes so
the design is reviewable as one coherent surface.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] One monotone, whitespace-heavy identity applied site-wide, dark theme as a first-class experience
- [ ] Reading serif for prose with authored casing preserved; lowercased mono for chrome and headings
- [ ] Navbar, hero, about, experience card, project card, skills, footer, post-list item, and status line composed from `blank` primitives
- [ ] Field-awareness lives in `persona`'s furniture; nothing persona-shaped is pushed into `blank`
- [ ] Cross-boundary tokens reach `blank`'s shadow chrome only via the shared base/tokens stylesheet
- [ ] A styleguide page renders the full furniture set in light and dark for review
