# 05 — `<blank-button>` (accessibility floor)

**What to build:** A `<blank-button>` action primitive with `primary`/`secondary`/`icon`
variants that renders a **real native `<button>`**, so keyboard operability,
Enter/Space activation, the button role, and native focus come for free. The `icon`
variant takes an SVG via a slot (no icon set is shipped) and accepts an `aria-label`,
passing it through to the real button so a consumer can supply an accessible name for
an icon-only action — the mechanism only; the value is the consumer's job. The base
stylesheet's rule that native focus is never suppressed without a replacement holds
here. This ticket carries the accessibility floor that is cheap now but expensive to
retrofit. Shadow DOM, tokens inherited through custom properties. Ships its
auto-registering entry and subpath export, regenerates the committed CEM, has a static
demo page, and is tested through its public interface in a real browser.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] `primary`/`secondary`/`icon` variants; renders a real native `<button>`
- [ ] Keyboard-operable with a visible native focus indicator (focus not suppressed)
- [ ] `icon` variant takes an SVG via a slot; no icon set shipped
- [ ] A supplied `aria-label` is forwarded to the real button on the `icon` variant
- [ ] Auto-registering entry, subpath export, regenerated CEM
- [ ] Static demo page (light/dark) and real-browser tests covering the a11y-floor behaviours
