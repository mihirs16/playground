# 06 — `<blank-stat>`

**What to build:** A `<blank-stat>` square stat tile — a big mono number and a label —
so a consumer can render metrics in the validated viz primitive. Shadow DOM, tokens
inherited through custom properties from the base sheet, internal structure
un-leakable. Ships its auto-registering entry and subpath export, regenerates the
committed CEM, has a static demo page, and is tested through its public interface in a
real browser.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] Renders a big mono number plus a label as a square tile
- [ ] Shadow DOM, tokens via custom properties; tests assert on observable output, not internal class names
- [ ] Auto-registering entry, subpath export, regenerated CEM
- [ ] Static demo page (light/dark) and real-browser public-interface tests
