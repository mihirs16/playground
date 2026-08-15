# 04 — Blog index

**What to build:** A blog index page that lets a reader browse what has been
written. It lists **exactly** the listed posts — generated over the same
`getStaticPaths()` slug set the permalink pages are built from, so it can never
drift out of sync with what was published. Each entry shows the post's title and,
when present, its description and metadata (reading time, tags, date), rendered
with the `post-list item` furniture so a reader can decide what to open. Unlisted
drafts never appear here.

**Blocked by:** 03.

**Status:** ready-for-agent

- [ ] Index lists exactly the listed posts, over the same slug set as the permalink pages
- [ ] Each entry shows title and, when present, description + metadata (reading time, tags, date)
- [ ] Unlisted drafts never appear in the index
- [ ] Rendered with the shared post-list item furniture
