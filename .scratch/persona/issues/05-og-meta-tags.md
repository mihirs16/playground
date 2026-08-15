# 05 — OG / meta tags in each page head

**What to build:** Per-post `<head>` metadata baked at build time so a post
shared into a chat app unfurls with a real preview and non-JS crawlers see
well-formed tags. Each post head carries its **title**; an **optional
`description`** meta tag that is **omitted entirely when the log has none** (an
absent preview beats a mangled auto-truncated one — `subtitle` is editorial and
must not be conscripted as the SEO summary); a **single site-wide default OG
image** (monotone card, no per-post image field); and standard boilerplate:
canonical URL, `og:type=article`, `og:site_name`, and
`twitter:card=summary_large_image`.

**Blocked by:** 03.

**Status:** ready-for-agent

- [ ] Each post head carries its title
- [ ] A log with a `description` emits the description meta tag; a log without one omits it entirely
- [ ] A single site-wide default OG image is set on every post; no per-post image field
- [ ] Canonical URL, `og:type=article`, `og:site_name`, and `twitter:card=summary_large_image` are present
- [ ] `subtitle` is not used as the SEO description
