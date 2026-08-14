# 07 — `rehype-raw` passthrough for live `blank` in log bodies

**What to build:** A log body can embed a live `blank` component (e.g. a post
about `blank` showing a real `<blank-stat>`), so writing about the library can
demonstrate it. `rehype-raw` (raw-HTML passthrough) is enabled in the markdown
pipeline; raw HTML is GFM-core-legal, so this is allowed grammar, not novel
syntax. The baked `<blank-*>` tag passes through to the served HTML (not escaped
or stripped) and the browser upgrades it at runtime. Raw-HTML passthrough is
normally an XSS vector and is safe here **only** under the **single-trusted-author
invariant** — this caveat is documented **both in the spec and in an inline
comment at the pipeline config**, so it travels with the code and must be
revisited first if authorship ever becomes multi-writer.

**Blocked by:** 03.

**Status:** ready-for-agent

- [ ] A raw `<blank-*>` tag in a log body passes through to the output rather than being escaped or stripped
- [ ] The passed-through element upgrades/hydrates in the browser at runtime
- [ ] The single-trusted-author invariant is documented inline at the pipeline config
