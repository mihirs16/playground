# 0001 — Starting point

**Date:** 2026-08-27

## Context
Mihir has a working pure-SVG scroll-driven exploded-axonometric prototype (built with
help) plus a concepts doc. He wants to learn the underlying skills bottom-up so he owns
the knowledge, not just the artifact.

## What we know about his level
- Comfortable reading code / running a repo (works in a multi-project playground).
- Has *seen* the advanced concepts (isometric projection, painter's algorithm, turntable
  rotation) but has NOT yet built up from SVG fundamentals — so treat foundations as new.
- Prefers step-by-step, explicit sequencing (his words: "step-by-step").

## Arc agreed
create SVGs → structure for rotation → rotate → animate. Each lesson ties back to
`CONCEPTS-exploded-axonometric.md`.

## Zone of proximal development
Started at Lesson 1 (coordinates + primitives) deliberately — even though the endpoint is
advanced, the rotation/animation skills are only durable if the coordinate mental model
(esp. y-grows-down, which explains `screenY = … − z`) is solid first.

## Open questions to calibrate later
- Does he know JS DOM APIs well? (affects how fast we can push into `createElementNS`.)
- Comfort with trig (sin/cos)? Critical for Lesson 3 rotation + eventual isometric math.
