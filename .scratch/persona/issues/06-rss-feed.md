# 06 — RSS 2.0 feed at `/rss.xml`

**What to build:** An RSS reader can subscribe without visiting the site. A
well-formed **RSS 2.0** feed is generated at build time at `/rss.xml` via
`@astrojs/rss` over the **same `getStaticPaths()` slug set** the index and pages
are built from — no new data source and no runtime tier, so the feed can never
drift from what was published. Each item reuses the post's title, description
(absent is fine), permalink, and date.

**Blocked by:** 04.

**Status:** ready-for-agent

- [ ] `dist/rss.xml` exists and is well-formed RSS 2.0
- [ ] Generated over the same listed-slug set as the index and permalink pages
- [ ] Each item carries title, description (omitted where absent), permalink, and date
