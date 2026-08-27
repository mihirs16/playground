# Concepts: Scroll-driven Exploded Axonometric (pure SVG)

A study doc for the prototype in `architecture-diagram.html`. It builds a rotating,
scroll-exploding isometric building diagram — the kind with towers floating above a
site plan, linked by dashed drop-lines — using **only SVG + vanilla JavaScript**.
No three.js, no anime.js, no WebGL, no Web Animations API.

The point of this file is to name every concept the prototype relies on, so each can
be taught/explored on its own later.

> **Interactive course:** each concept below is taught hands-on in the sibling
> teaching workspace [`../learn-svg/`](../learn-svg/index.html) — 7 dark-themed lessons
> with live sliders/demos (coordinates → rotation → isometric projection → animation →
> painter's algorithm → a scroll-driven capstone). Start at `learn-svg/index.html`.

---

## 1. Axonometric (isometric) projection — faking 3D with 2D math

The whole illusion of depth comes from **one function** that maps a 3D grid point
`(x, y, z)` to a 2D screen point `(screenX, screenY)`:

```js
screenX = OX + (x - y) * A
screenY = OY + (x + y) * B - z
```

- `x` and `y` are positions on the ground plane; `z` is height.
- `A = S*cos(30°)`, `B = S*sin(30°)` encode the **viewing angle** (classic 30° isometric).
- `(x - y)` drives horizontal position; `(x + y)` drives how "far back/down" a point sits;
  subtracting `z` lifts things up the screen.

Key idea: this is an **orthographic / parallel** projection — parallel lines stay
parallel, nothing converges to a vanishing point. That's what makes it read as a
technical drawing rather than a photograph.

**Teach-worthy sub-concepts**
- Orthographic vs. perspective projection (why iso has no vanishing point).
- Isometric vs. dimetric/trimetric (what changing `A`/`B` does).
- Why the angle constants are `cos`/`sin` of the projection angle.

---

## 2. Drawing solids as flat polygons (painter's algorithm)

There is no 3D engine deciding what's in front. Each box (`prism()`) is drawn as a
handful of `<polygon>`s — a top face plus side faces. Depth is faked by:

1. **Painter's algorithm** — draw far things first, near things last, so nearer shapes
   paint over farther ones. Both whole objects and the faces *within* a box are sorted
   by depth before drawing.
2. **Flat shading** — each face gets a slightly different fill (`top` lightest, sides
   darker) to imply light without any real lighting model.

### ⚠️ Pitfall: hard-coded face selection breaks under rotation

The **first** version of this prototype took a shortcut: at a fixed isometric angle a
box only ever shows **three** faces (top, `+x`, `+y`), so it drew *only those three*.
This is a static, baked-in form of **back-face culling** — and it's a trap the moment
anything rotates.

Once the turntable (see §5) turned past ~90°, the faces that were previously hidden
(the `-x` / `-y` backs) swung toward the viewer — but they were **never drawn**, so the
boxes looked **hollow / missing faces**, as if you were seeing through them.

**Two ways to fix it:**

- **Draw all four side faces, back-to-front** (what this prototype now does). Simple and
  correct at every angle; the cost is drawing a couple of hidden faces that get
  immediately overpainted. The per-box sort uses `depthAt()` on each face's centre, then
  the top face is drawn last (it's always the frontmost face when viewed from above).
- **Recompute back-face culling per frame** — only draw a face whose *rotated* outward
  normal points toward the viewer. Fewer polygons, but more code.

**The lesson:** any visibility optimisation that assumes a fixed camera (which faces are
"front", which objects occlude which) must be **recomputed whenever the view changes**,
or it silently produces holes. Culling and depth-sort are *view-dependent*.

**Teach-worthy sub-concepts**
- The painter's algorithm and its failure cases (why a z-buffer exists).
- Back-face culling — and why a *static* culling assumption breaks under rotation.
- View-dependent vs. view-independent computations (what must be redone per frame).
- Fake shading / ambient occlusion by hand-picked fills.

---

## 3. Composition & content structure

The scene is assembled from small reusable helpers, each emitting SVG:
`prism()` (extruded box), `slab()` (thin plate), `tree()`, `route()` (polyline path),
`drop()` (dashed vertical connector), `storeys()` (floor striations on a tower face).

The building is organised into **stacked layers** at increasing heights
(`Zg` ground → `Zp` podium → `Zu` upper deck → `Zt` towers), tied together by the
vertical drop-lines. This layered decomposition is the "exploded view" idea.

**Teach-worthy sub-concepts**
- Building complex vector art from small primitive-emitting functions.
- SVG basics used here: `<polygon>`, `<polyline>`, `<line>`, `<circle>`, `<ellipse>`,
  `<pattern>` (hatching), `stroke-dasharray` (dashed lines).
- Generating DOM/SVG nodes in JS via `createElementNS` (SVG needs the XML namespace).

---

## 4. Scroll-driven animation (no animation library)

Animation = **re-running `draw(spread)` every frame with a changing number**.

- **Layout:** a tall `#scroll` runway (`360vh`) with a `position: sticky` stage, so the
  drawing stays pinned to the viewport while the page scrolls "past" it.
- **Progress:** scroll position is converted to a `0→1` value from the runway's
  bounding rect.
- **Easing:** that value passes through `smoothstep` (ease-in-out), then a **lerp
  follow** (`cur += (target - cur) * 0.12`) so motion glides instead of snapping to the
  raw scroll value each frame.
- **Loop:** a `requestAnimationFrame` loop redraws when `cur` is still catching up.

The `spread` value (0 = collapsed stack, 1 = fully exploded) scales every layer's
height, so scrolling pulls the layers apart along the drop-lines.

**Teach-worthy sub-concepts**
- `position: sticky` + a scroll runway as a scroll-timeline technique.
- Mapping/normalising scroll position to a 0–1 progress value.
- Easing functions (smoothstep) and lerp-based "smooth follow" / critically-damped feel.
- `requestAnimationFrame` render loops vs. CSS transitions vs. WAAPI.
- Immediate-mode ("clear and redraw everything") vs. retained-mode ("build once,
  update transforms") rendering — this prototype uses immediate mode for simplicity.

---

## 5. Turntable rotation about the vertical axis

To spin the whole massing while it explodes, `(x, y)` is **rotated about the site
centre before projecting**:

```js
rx = CX + dx*cos(ROT) - dy*sin(ROT)
ry = CY + dx*sin(ROT) + dy*cos(ROT)   // dx,dy = point relative to centre
// then feed (rx, ry, z) into the isometric projection
```

`ROT` is also driven by scroll, so explode + rotate are a single gesture.

Because rotation changes which box is nearest, the painter's-order sort is redone
against **rotated depth** (`rotDepth()`) each frame.

**The key boundary this demonstrates:** a **single vertical-axis** turntable is just a
2D rotation applied before projection — cheap, and fully doable in SVG. You only need
three.js when you want *more*: free tumbling on a second axis (hand-sorted painter's
order breaks → you need a real z-buffer), true perspective, lighting/shadows, or
loading real 3D models.

**Teach-worthy sub-concepts**
- The 2D rotation matrix and rotating about an arbitrary pivot (translate → rotate → translate back).
- Order of operations: rotate in model space, *then* project.
- Why depth-sorting must be recomputed after rotation.
- The concrete SVG-vs-three.js decision boundary (single-axis spin OK; multi-axis / perspective / lighting → 3D engine).

---

## Quick map: concept → where to look in the code

| Concept | Look for |
| --- | --- |
| Isometric projection | `P(x,y,z)` |
| Turntable rotation | `ROT`, the rotation inside `P`, `rotDepth()` |
| Solids as polygons + painter's order | `prism()`, the `.sort(...)` on `towers` |
| Reusable SVG primitives | `prism/slab/tree/route/drop/storeys` |
| Layered "exploded" structure | `Zg/Zp/Zu/Zt`, `draw(spread)` |
| Scroll → progress → animation | `#scroll` + sticky `#stage`, `progress()`, `loop()` |
| Easing / smoothing | `smooth()`, the lerp `cur += (target-cur)*0.12` |

---

## What is deliberately NOT here (and would change the tooling)

- **Perspective projection** (vanishing points) — needs a divide-by-depth; would move
  you toward a real 3D pipeline.
- **Multi-axis free rotation** — breaks the hand-sorted painter's order; wants a z-buffer (three.js).
- **Lighting, shadows, materials, real 3D models** — three.js / WebGL territory.
- **Retained-mode performance** — this redraws all SVG every frame; production would
  build nodes once and animate transforms (and could use WAAPI for the tweening).
