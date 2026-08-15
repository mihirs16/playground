# 03 — `<blank-stack>` and `<blank-card>` layout primitives

**What to build:** The two content-blind layout/container primitives. `<blank-stack>`
vertically flows its children with token spacing selected by a `space` attribute —
pure spacing, with no heading or icon slots (a titled section stays a composition the
consumer owns, not a `blank` concern). `<blank-card>` is a padded box with a single
default slot and no fields or named slots, so a consumer can wrap arbitrary content in
the standard padded surface. Both use Shadow DOM, share one constructed stylesheet via
`adoptedStyleSheets`, keep their internal class names un-leakable, and inherit tokens
through custom properties from the base sheet. Each ships its auto-registering entry
and subpath export, regenerates the committed CEM, has a static demo page, and is
tested through its public interface in a real browser.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] `<blank-stack>` `space` attribute maps to token spacing → asserted computed gap
- [ ] `<blank-stack>` has no heading/icon slots
- [ ] `<blank-card>` is a padded box with a single default slot, no named slots
- [ ] Both use Shadow DOM and inherit tokens via custom properties; internal class names not asserted on
- [ ] Auto-registering entries, subpath exports, and regenerated CEM
- [ ] Static demo pages (light/dark) and real-browser public-interface tests
