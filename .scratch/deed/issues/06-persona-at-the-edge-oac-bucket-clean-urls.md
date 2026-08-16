# 06 — Persona at the edge: private OAC bucket, clean-URL CloudFront Function & path routing

**What to build:** Persona's blog pages served through the edge with clean URLs,
off the single distribution from 05. `deed` provisions the private persona S3
bucket locked to CloudFront via OAC (no public S3 website endpoint, so clients
cannot bypass CloudFront to hit the bucket directly), a `deed`-owned CloudFront
Function rewriting `/logs/<slug>/` → `…/index.html` so persona's directory-format
clean URLs resolve against that private bucket, and the path behavior on the
distribution routing `/logs/<slug>/` to the persona bucket (the default behavior
stays the custodian API origin). `deed` owns the distribution, the function, and
the bucket; the origins' *content* is not `deed`'s.

Media is **not** routed here: it is served CDN-direct from its own distribution
(ADR-0002, deed ticket 09), never through the custodian origin.

**Blocked by:** 05.

**Status:** ready-for-agent

- [ ] Private persona S3 bucket provisioned and OAC-locked to CloudFront (no public-website endpoint)
- [ ] `deed`-owned CloudFront Function rewrites `/logs/<slug>/` → `…/index.html`
- [ ] Distribution path behavior routes `/logs/` to the persona bucket; default stays the custodian API origin
