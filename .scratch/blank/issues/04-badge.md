# 04 — `<blank-badge>`

**What to build:** A `<blank-badge>` that renders text plus an optional ink state dot
selected by an attribute (`active`/`idle`), so a consumer can render a live-status
surface as an instance of it rather than a bespoke component. The dot animates on state
change, and the animation is gated on `prefers-reduced-motion` so the motion is a
nice-to-have that never fights accessibility. Shadow DOM, tokens inherited through
custom properties, internal structure un-leakable. Ships its auto-registering entry and
subpath export, regenerates the committed CEM, has a static demo page, and is tested
through its public interface in a real browser.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] Renders text plus an optional ink state dot driven by an `active`/`idle` attribute
- [ ] Dot animates on state change
- [ ] Animation suppressed under `prefers-reduced-motion`
- [ ] Shadow DOM, tokens via custom properties; tests assert on observable output, not internal class names
- [ ] Auto-registering entry, subpath export, regenerated CEM
- [ ] Static demo page (light/dark) and real-browser public-interface tests
