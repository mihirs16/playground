# Rotating SVG — a teaching workspace

An interactive, self-contained course teaching Mihir how to build the pure-SVG
scroll-driven **exploded axonometric** in `../prototypes/architecture-diagram.html`,
bottom-up. Created via the `mattpocock-skills:teach` skill.

**Start here:** open [`index.html`](./index.html) in a browser (course table of contents).

## Layout (teach-skill convention)
| Path | What it is |
| --- | --- |
| `MISSION.md` | Why Mihir is learning this — grounds every lesson. Read first. |
| `NOTES.md` | His learning preferences (dark theme, gentle JS, trig refreshers). |
| `RESOURCES.md` | High-trust sources (mostly MDN) cited in lessons. |
| `index.html` | Course TOC linking all lessons + reference. |
| `lessons/*.html` | The 7 lessons. Self-contained, dark-themed, each with a live demo + quiz. |
| `reference/*.html` | Durable cheat sheets (revisited; lessons are not). |
| `assets/course.css` | **Shared** stylesheet — every lesson links it. Change the palette here, all lessons update. |
| `learning-records/*.md` | ADR-style log of what was taught & why; drives the next session's ZPD. |

## The arc (all 7 built, course complete)
1. Drawing with coordinates (primitives, y-down)
2. viewBox & rotation-ready art (pivots, groups)
3. Rotate it (sin/cos, 2D rotation formula)
4. Isometric projection (`P(x,y,z)`, interactive sliders)
5. Animate it (rAF, smoothstep, lerp follow)
6. Solid faces & painter's algorithm (the hollow-box bug)
7. Capstone — scroll-driven exploded scene (mini-prototype)

Each lesson maps to a section of `../prototypes/CONCEPTS-exploded-axonometric.md`.

## For a future agent picking this up
- Read `MISSION.md`, `NOTES.md`, and the newest `learning-records/*.md` to find where he is.
- Reuse `assets/course.css` and existing patterns (`.figure`, `.tryit`, `.quiz`) — don't
  reinvent; extend the component library instead.
- The course is "complete" but open-ended: likely next moves are guided build exercises
  (add drop-lines / a 4th tower), a line-by-line diff against the real prototype, or a
  brand-new topic. Ask him; he likes driving the curriculum and learning by playing.
- Quizzes keep all answers the same length (no formatting tells) — preserve that.
