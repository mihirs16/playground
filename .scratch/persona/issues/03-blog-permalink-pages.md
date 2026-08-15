# 03 — Blog permalink pages: prose baked into real files

**What to build:** The tracer bullet that realizes the whole reason `persona`
exists. At build time, `getStaticPaths()` fetches `custodian`'s slugs and bakes
every **listed** log into a real prerendered file at `/logs/<slug>/` (emitted as
`/logs/<slug>/index.html`), with the post prose present in the initial HTML
payload — so a shared or bookmarked permalink resolves as a real page, survives a
hard refresh, paints instantly, and reads fully with JavaScript disabled.
Markdown-to-HTML is a **build step** via Astro's own remark/rehype pipeline
(`<blank-markdown>` is not used); prose is emitted as light-DOM semantic HTML
styled by `blank`'s global base. An **unlisted** draft is baked to a reachable
file at its exact slug but never appears in any index. A build against a **down
`custodian` fails loudly** rather than shipping missing content. `persona` owns
this surface's accessibility floor: correct heading order, alt text, real `<a>`
links, and no suppression of native focus.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] Each listed slug produces a real file at `dist/logs/<slug>/index.html`; prose is in the served HTML, not an empty shell
- [ ] Permalinks survive a hard refresh; there is no client-side router or client navigation
- [ ] Markdown renders via Astro's remark/rehype build step to light-DOM semantic HTML styled by `blank`'s base
- [ ] Unlisted slugs produce a reachable file but never appear in any index
- [ ] A build against a down/erroring `custodian` fails loudly
- [ ] Heading order, alt text, real `<a>` links, and native focus are all correct (a11y floor)
