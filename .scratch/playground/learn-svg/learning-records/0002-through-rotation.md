# 0002 — Progress through rotation (Lessons 1–3)

**Date:** 2026-08-27

## Covered
- **L1** coordinates & primitives (y-grows-down → explains `screenY = … − z`).
- **L2** viewBox re-homes the origin; `<g>` groups; rotation is *about a pivot*
  (plain `rotate()` = origin pivot → "flies away" bug).
- **L3** sin/cos refresher (unit circle, radians = deg·π/180), the 2D rotation formula
  `rx = dx·cosθ − dy·sinθ`, the translate→rotate→translate pivot dance, and the key
  distinction: **rotate in model space THEN project** (why the prototype hand-computes
  rotated coords instead of using `transform="rotate"`).

## Design choices honoured
- Dark theme (his request); JS introduced gently, each L3 slider handler commented.
- Trig refresher embedded in L3 per his "half-remembered" self-rating.

## Next (ZPD)
- **L4 Animate it:** `requestAnimationFrame` loop, time-based motion, `smoothstep`
  easing, and the lerp "smooth follow" (`cur += (target−cur)*0.12`). Then optionally
  **L5:** scroll → progress → drive rotation+spread (the full prototype gesture).
- Watch: does he want to attempt "Way B" (manual coordinate rotation in JS) as a hands-on
  exercise? He was offered it. If yes, that's a good pre-L4 skills drill.
