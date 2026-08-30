# Concepts: mihirsingh.dev as a system diagram (graph portfolio)

A study doc for the prototype in `mihirsingh.dev-graph-portfolio.html`. It reimagines the
homepage as a **directed graph of the person** — identity at the root, typed edges as verbs
(*i am / working at / built / wrote*), nodes as the things — rendered in the existing
mihirsingh.dev visual language.

The design thesis being tested: **visual clarity = a correctness signal**. If the homepage
*is* the system diagram of the person, a dangling node or an edge that doesn't resolve reads
as a bug the moment it's on screen — the medium embodies the philosophy instead of just
describing it.

Source of the shape: `mihirsingh.dev-diagram-concept.excalidraw` (the proof sketch). The rule
we set early: *if the idea can't be drawn as a clean diagram, it isn't worth shipping.* It drew
clean, so we built it.

> Companion: this prototype deliberately **retains the real site's design system** (see §1) so
> it reads as an evolution of mihirsingh.dev, not a generic redesign.

---

## 1. Extract the design system before designing anything

The single most useful step was **not** inventing a look — it was recovering the live site's
system from its compiled output, then building inside it. The first pass ignored this and
produced a generic "AI blueprint" look (blue accent, rounded cards, drop shadows, grid
background) that had to be thrown away.

How the real tokens were recovered:

```
curl -sL https://mihirsingh.dev -o site.html           # grab the HTML
grep -oE '(href|src)="[^"]*\.css"' site.html            # find the compiled stylesheets
curl the .css files, then:
  grep font-family   → Roboto Mono (mono everywhere)
  grep '#[0-9a-f]{3,8}' | sort | uniq -c   → the real palette
```

The recovered system:

| Token        | Value            | Notes                                   |
|--------------|------------------|-----------------------------------------|
| type         | `Roboto Mono`    | monospace for *everything*              |
| ground / ink | `#fff` / `#000`  | pure black on pure white                |
| greys        | `#5f5f5f`, `#969696`, `#5c5c5c` | body / footer / skills    |
| accent       | *(none)*         | strictly black-and-white                |
| borders      | `1px solid #000` | **no radius, no shadow**                |
| signature 1  | `h3::after`      | a hairline rule that fills the row after a heading |
| signature 2  | experience cards **invert to black-on-hover** (`.altContent{color:#fff}`) |

**Learning:** a portfolio's identity lives in its *constraints* (mono, B&W, hairline rules,
no accent), not its content. Reuse the constraints and the new thing feels like the same
author. The hairline-after-heading and the hover-invert became the two motifs we leaned on.

---

## 2. Topology is the content — read the diagram literally

The Excalidraw isn't a left-indented tree (the first mistake) — it's a **centre spine with two
full-height side rails**:

```
● Mihir Singh
├─ i am ──────► about
│               └─ working at ──► Emu ──► previously ──► [wtv][qmul][kloudrac]
├─ built ─────► [ project rows ]      (LEFT rail, forks near root, hooks in at the bottom)
└─ wrote ─────► [ tall cards  ]      (RIGHT rail, forks near root, drops in via a manifold)
        now ► [ steam ] [ commits ]  (floating "now" tiles)
```

Two edge *types*, which turned out to demand two different label behaviours (§4):
- **Spine edges** — short, between stacked sections, centred on the column.
- **Rail edges** — long, running the whole page height down the outer margins.

**Learning:** when the shape is the spec, honour it exactly — including the *asymmetry*
(built forks left, wrote forks right, at the **same** height). Flattening it to a tidy tree
lost the whole point.

---

## 3. Draw the connectors from measured geometry, not hand-authored paths

The elbow arrows are a single absolutely-positioned `<svg class="wires">` overlay. Nothing in
it is hard-coded: on every layout it **measures the boxes** and recomputes orthogonal paths.

```js
function anchors(id){                        // box edge-points in stage-local coords
  const s = stage.getBoundingClientRect(), r = el.getBoundingClientRect();
  return { t:{...}, b:{...}, l:{...}, r:{...} };  // top/bottom/left/right centres
}
// spine:      M root.b → about.t        (arrowed)
// previously: M emu.l → left → down → prev[0].l   (elbow)
// left rail:  M spine → Xl → down; then one arrow hooking into each row
// right rail: M spine → Xr → down → manifold across the card-tops; drop-arrows into each
```

Recompute on `load`, `resize`, `ResizeObserver(document.body)`, and a couple of timed passes
(fonts settling). `svg{ overflow:visible }` lets the rails sit outside the stage box.

**Learnings:**
- **Measure, don't hardcode.** One `anchors()` helper + path strings beats fragile SVG `d`
  attributes, and it survives any content/viewport change — which is what makes it
  *data-driven* in spirit even though the boxes are in normal flow.
- **Push rails to the viewport, not the stage.** To get the long edges running through real
  negative space, the rail X is computed from `clientWidth` (`inset = clamp(26, 5vw, 84)`)
  and converted to stage-local (`x - stage.left`), so the rails hug the window edges no matter
  how narrow the centre column is.
- **Arrowheads via one `<marker>`**, `orient="auto-start-reverse"`, `fill-opacity` matched to
  the line so nothing shouts.

---

## 4. Two label behaviours, both pure CSS `position: sticky`

The verb labels are the trickiest interaction, and both variants are **zero-JS** — just sticky
anchoring:

**Spine verbs (`i am`, `working at`, `previously`) — preview at the bottom.**
The *upcoming* section's title rides up from the bottom of the viewport and scrolls off as you
enter its section. That's `position: sticky; bottom: 22px` on a label placed at the *top* of a
tall section: while the section enters from below, its top-anchored label is clamped to the
viewport bottom; once you scroll past, it releases upward.

**Rail verbs (`built`, `wrote`) — ride the rail end-to-end.**
Each label lives in an absolutely-positioned "rail zone" whose `top`/`height` are set by
`draw()` to span exactly fork → terminus. Inside, `.rl { position: sticky; top: 78px }` keeps
it pinned on the rail for the entire span, releasing only at the section.

```js
zone("rz-built", Xl, branchY-12, bBot   - branchY + 24);  // spans the whole left rail
zone("rz-wrote", Xr, branchY-12, manY   - branchY + 24);  // spans the whole right rail
```

**Learnings:**
- **`sticky bottom` ≠ `sticky top`.** Anchor side changes *which* moment the label is pinned.
  "Preview the next thing" is a bottom-anchor; "label this whole span" is a top-anchor inside a
  measured wrapper. Naming the desired moment first ("visible at the bottom of the previous
  section") tells you the anchor.
- **Match the label mechanism to the edge type.** Short spine edges → bottom-preview; long rail
  edges → span-sticky. One size did not fit both.
- Every label carries a `background: var(--paper)` so it **masks the edge-line** it sits on —
  the label *is* the gap in the wire, at every scroll position.

---

## 5. Give each section a viewport to breathe in

The real site centres one section at a time in a sea of whitespace. Reproduced with
`min-height: 86vh; justify-content: center` per `.grp` on desktop (`92vh` for the last). The
long rail-edges then run as quiet lines *through* that negative space — the emptiness becomes
the canvas the diagram is drawn on, not wasted room.

**Learning:** negative space and the diagram are the same feature here. Once sections were
viewport-tall, the rails stopped looking like plumbing and started looking like structure.

---

## 6. Box = node, line = edge — don't dissolve the grammar

When we started stripping outlines to play with negative space, the right cut was the **about
prose** (it's a *description*, not a node) — border removed, hover-invert removed, left to float.
But the job/project/writing cards kept their `1px` borders on purpose: in a diagram a **box is a
node and a line is an edge**, so an arrow must terminate on something bordered. Strip every
outline and the rails would point at floating text and the diagram loses its grammar.

**Learning:** negative-space minimalism has a floor set by the diagram's own semantics. Fade the
*edges* (opacity `0.42`), open the *space*, but keep the *nodes* legible so the edges have
something to land on.

---

## 7. Responsive: collapse the 2D diagram to a 1D document

Below `760px` the whole SVG layer is hidden and the centred canvas becomes a single stacked
column, with the verb labels falling back to the site's left-aligned `header ─────` hairline
style. Same DOM, same content, same data — only the connector layer and label styling switch.

**Learning:** a 2D flowchart can't reflow; it has to *degrade to a list*. Designing the DOM in
readable top-to-bottom order from the start made the mobile fallback nearly free.

---

## Gotchas logged along the way

- **Cascade order bit us.** A desktop `.vh` override placed in an early `@media` block was
  silently beaten by the base `.vh` rule defined *later* (equal specificity → later wins), which
  left-aligned the labels. Fixed by raising specificity (`.col .vh`) — but the real lesson is
  **put media-query overrides after the base rule they override**.
- **Google Fonts is blocked by the artifact CSP**, so `Roboto Mono` falls back to system mono in
  the preview. On the real deploy: self-host the face. The prototype is not pixel-exact on type.
- **`Date.now()`/`Math.random()` are unavailable** in some sandboxes; the "commit bars" heights
  are generated from the index (`(i*37)%78`) rather than randomness — deterministic by design.

---

## Open threads (not yet built)

- **Scroll-driven edge-drawing** — have each edge *trace in* (dash-offset) as you approach its
  target, instead of drawing all at once.
- **A real node/edge data schema** — the boxes are still hand-authored HTML. The end state is a
  `{nodes, edges}` document that both this renderer and the Penpot design read from, so *adding
  a project is a data edit*. This is the bridge to production.
- **Content mapping is provisional** — built→project rows, wrote→tall cards follows the drawn
  geometry; swap if projects should get the richer cards.
- `wrote` is placeholder (no blog yet); keep the edge or cut it until there's writing.
