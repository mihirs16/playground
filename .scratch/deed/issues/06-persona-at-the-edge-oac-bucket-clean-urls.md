# 06 — Persona at the edge: private OAC bucket, clean-URL CloudFront Function & path routing

**What to build:** Persona's blog pages served through the edge with clean URLs,
and media routed to the custodian origin, all off the single distribution from
05. `deed` provisions the private persona S3 bucket locked to CloudFront via OAC
(no public S3 website endpoint, so clients cannot bypass CloudFront to hit the
bucket directly), a `deed`-owned CloudFront Function rewriting `/logs/<slug>/` →
`…/index.html` so persona's directory-format clean URLs resolve against that
private bucket, and the path behaviors on the distribution that namespace the
edge by prefix — `/media/<key>` to the custodian origin, `/logs/<slug>/` to the
persona bucket. `deed` owns the distribution, the function, and the bucket; the
origins' *content* is not `deed`'s.

**Blocked by:** 05.

**Status:** ready-for-agent

- [ ] Private persona S3 bucket provisioned and OAC-locked to CloudFront (no public-website endpoint)
- [ ] `deed`-owned CloudFront Function rewrites `/logs/<slug>/` → `…/index.html`
- [ ] Distribution path behaviors route `/media/` to the custodian origin and `/logs/` to the persona bucket
