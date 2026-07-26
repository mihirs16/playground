# persona: blog delivery, crawlability and permalinks

Type: grilling
Status: open
Blocked by: 09, 15

## Question

Blogs load from custodian at runtime. What does a crawler, a link preview scraper, and a reader with a direct permalink actually get?

- **Crawlability** — client-fetched blog bodies mean the static HTML is an empty shell. Search engines execute JavaScript inconsistently and with delay; most link-preview scrapers (Slack, iMessage, WhatsApp, Mastodon, LinkedIn) execute none at all. A shared blog post would preview as a blank page. Is that acceptable, or does it need solving?
- **Per-post OG tags** — these must be in the initial HTML response to work at all. Purely client-side rendering cannot produce them. If per-post previews matter, something has to pre-render or inject them.
- **Permalinks** — does `/blog/some-post` resolve as a real URL on a static host, or only via client-side routing from the index? Direct navigation and hard refresh have to work.
- **Options if crawlability does matter**: build-time prerender of published posts with a runtime refresh for edits; an edge worker that fetches from custodian and injects meta tags; ISR-style regeneration on a webhook from custodian; or serving persona from a small server after all.
- **Empty and error states** — what renders when custodian is slow or down? A visitor arriving at a blog permalink during an outage sees nothing, unlike a static site.
- **RSS/Atom feed** — generated where? A feed is build-time by nature, which sits awkwardly with runtime-loaded posts.
- **Does any of this change the runtime-loading decision**, or is it accepted with mitigations?

## Context

Blocked on `09` (the read contract) and `15` (the framework determines which mitigations are even available).

This ticket exists because the charting session flagged a tension rather than resolving it. You chose runtime loading for blogs with the reasoning that custodian needs health monitoring regardless — which addresses availability, but not crawlability or link previews. Those are separate failure modes and neither is fixed by monitoring.

Stated plainly so a later session doesn't have to rediscover it: markdown blogs are the content type where crawlability and share previews matter most, and runtime loading is the delivery model that serves them worst. That may still be the right trade for a personal site — but it should be a decision, not an accident. `11` covers the availability half.
