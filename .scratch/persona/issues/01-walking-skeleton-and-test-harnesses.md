# 01 — Walking skeleton: Astro SSG + both test harnesses

**What to build:** A booting `persona` Astro project that stands up the whole
shape the rest of the work hangs off, with nothing business-facing yet. Astro is
configured as a pure static generator (`output: 'static'`, `build.format:
'directory'` for clean `/logs/<slug>/` URLs, no client-side router). It takes a
workspace dependency on the local `blank` (`workspace:*`) and imports it **only
through `blank`'s published `exports` map** — never a deep `../blank/src` reach.
`blank`'s base / tokens stylesheet is imported once, globally, in the root
layout; `blank`'s auto-registering `@customElement` entry is loaded in a
client-side script so the browser upgrades `<blank-*>` tags (no `@lit-labs/ssr`).
Both test seams are established against a trivial placeholder page: Seam 1 runs
the real Astro build against a **faked `custodian` `/v1`** HTTP surface and
asserts on the emitted `dist/`; Seam 2 mounts a custom element in a real browser
with a faked `fetch`.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] Astro builds with `output: 'static'` and `build.format: 'directory'`; no client-side router
- [ ] Workspace dependency on local `blank`, imported only through its published `exports` (no deep `src` import)
- [ ] `blank` base/tokens stylesheet imported once in the root layout; auto-registration entry loaded in a client-side script
- [ ] Seam 1 harness: Vitest drives the real Astro build against a faked `custodian /v1` (stub/MSW) and asserts on `dist/`
- [ ] Seam 2 harness: `@open-wc/testing` on Web Test Runner + Sinon `fetch` stub mounts an element in a real browser
- [ ] A trivial page proves the build → fake-`custodian` → assert-`dist` loop end-to-end
