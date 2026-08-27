# 0004 — Core arc complete (Lessons 6–7)

**Date:** 2026-08-27

## Built
- **L6 Solid faces & painter's algorithm** — filled cube, far→near face sort, flat shading;
  interactive reproduction of the "hollow boxes past 90°" bug via a "cull to 3 faces"
  toggle + rotation slider. Covers CONCEPTS §2 (view-dependent culling, SVG-vs-three.js
  boundary).
- **L7 Capstone** — a real scroll-driven exploded axonometric: 3 stacked boxes that rotate
  and explode on scroll, using sticky runway + progress + smoothstep + lerp follow +
  painter's sort. Assembles L1–L6. Points him back to the real prototype to read.
- **index.html** — course TOC. **README.md** — orientation for future agents.
- Cross-linked CONCEPTS doc → the course.

## State
All 7 planned lessons exist; MISSION arc marked complete. Course committed to git
(previously the whole `.scratch/playground/` tree was untracked).

## Where a future session goes
Not more core lessons — growth is now by *doing*:
- Guided build exercises (add drop-lines / 4th tower / reverse rotation) — offered in L7.
- Line-by-line diff of his understanding vs the real `architecture-diagram.html`.
- Per-frame back-face culling as an optional deepening (offered in L6).
- Or a brand-new topic. He drives the curriculum; keep offering concrete branches.

## Durable prefs (carry forward)
Dark theme; learns by playing with live demos; gentle JS with commented handlers;
trig refreshers when math appears; quizzes keep equal-length answers.
