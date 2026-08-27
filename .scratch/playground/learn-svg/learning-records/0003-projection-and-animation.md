# 0003 — Projection & animation (Lessons 4–5)

**Date:** 2026-08-27

## Change to the arc
Mihir asked for a hands-on isometric slider and whether projection was its own lesson.
→ **Inserted L4 "Isometric projection"** (was skipped in the original create→rotate→animate
arc). Animation moved to **L5**. Mission arc updated. Good call on his part — projection is
§1 of CONCEPTS and the core illusion.

## Covered
- **L4** the `P(x,y,z)` formula (`screenX=(x−y)A`, `screenY=(x+y)B−z`), A/B=S·cos/sin(angle),
  orthographic = no vanishing point. Interactive wireframe cube with angle/rotation/height
  sliders; combined L3 rotation (rotate-then-project). Kept wireframe on purpose to defer
  painter's algorithm.
- **L5** rAF render loop, frame-rate vs time-based caveat, smoothstep easing, and the
  lerp smooth-follow `cur += (target−cur)*k`. Two live demos: auto-spin + target-follow with
  adjustable k. Tied explicitly to CONCEPTS §4 and scroll-driving.

## He now has all pieces to READ the full prototype except:
- **Painter's algorithm / solid faces + culling pitfall** (§2) → planned **L6**.
- Scroll plumbing (sticky runway → progress) — briefly covered conceptually in L5; could be
  a short standalone or folded into a capstone.

## Next (ZPD) — offered choices
L6 solid faces + painter's algorithm (the "hollow boxes under rotation" bug) is the natural
next and completes CONCEPTS coverage. Also offered: wire cube to real scroll; or a capstone
rebuild of the mini-prototype from scratch (best storage-strength test).

## Note
Prefers learning-by-playing (sliders, live demos) and driving the curriculum himself —
keep offering concrete "want X next?" branches rather than a fixed track.
