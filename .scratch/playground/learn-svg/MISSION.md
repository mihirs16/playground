# Mission: Build rotating, animated SVG scenes from scratch

## Why
Mihir has a prototype — a scroll-driven **exploded axonometric** building diagram
(`.scratch/playground/prototypes/architecture-diagram.html`) built with **pure SVG +
vanilla JS** (no three.js, no anime.js). He wants to genuinely *understand* how it works
by learning the underlying skills bottom-up, so he can build things like it himself.

## The learning arc
1. **Create SVGs** — shapes by hand, canvas & coordinate system. ✅ (L1)
2. **SVGs built for rotation** — viewBox, groups, pivots. ✅ (L2)
3. **Rotate them** — 2D rotation math (sin/cos, pivot dance). ✅ (L3)
4. **Isometric projection** — the `P(x,y,z)` formula; rotate-then-project. ✅ (L4)
   *(inserted at Mihir's request — projection is its own core concept, §1 of CONCEPTS)*
5. **Animate them** — `requestAnimationFrame`, easing (smoothstep), lerp follow. ✅ (L5)
6. **Solid faces & painter's algorithm** — §2 of CONCEPTS, the culling pitfall. ✅ (L6)
7. **Capstone** — scroll-driven exploded axonometric, all techniques assembled. ✅ (L7)

**Status: all 7 lessons built — the core arc is complete.** Remaining growth is by
building (guided exercises, diffing against the real prototype) or a new topic.

Endpoint: he can read `architecture-diagram.html` and its companion
`CONCEPTS-exploded-axonometric.md` and understand every concept — isometric projection,
painter's algorithm, turntable rotation, scroll-driven rAF loops.

## Grounding
Every lesson should connect back to a concept in `CONCEPTS-exploded-axonometric.md`.
That doc is the map; these lessons teach the territory one skill at a time.

## Definition of done
Not "can recognise the code" (fluency) but "can build a small rotating, animated SVG
scene unaided, and explain why each piece works" (storage strength).
