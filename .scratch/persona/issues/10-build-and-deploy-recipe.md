# 10 — Build & deploy recipe: publish as one gesture

**What to build:** Publishing a batch of posts is one deliberate gesture. A
single root `just` / deploy recipe builds the whole static site and then does the
laptop `s3 sync` of the output to the private S3 bucket plus a CloudFront
invalidation, under SSO creds — no long-lived render tier, no webhook, no timer.
The build fetches all baked content (logs, index, OG, RSS, profile) from
`custodian`'s public `/v1` with no admin token, and a failed `custodian` fetch
**fails the build loudly** so a site with silently missing content is never
shipped. `broom` and `custodian` stay completely unaware of `persona` — the
coupling is one-directional and build-time only. (Provisioning the bucket,
CloudFront, OAC, and the `/logs/<slug>/` → `index.html` rewrite Function is
`deed`'s work — assumed here as the contract the output depends on, not built.)

**Blocked by:** 03, 04, 05, 06, 07, 08, 09.

**Status:** ready-for-agent

- [ ] A single `just`/deploy recipe builds the site and runs `s3 sync` + CloudFront invalidation under SSO creds
- [ ] Build fetches baked content from `custodian`'s public `/v1` with no admin token
- [ ] A failed `custodian` fetch during the build fails loudly
- [ ] No `custodian`→`persona` webhook or timer; the coupling stays one-directional and build-time only
