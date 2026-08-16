# ADR-0002: Media served CDN-direct, not through the custodian origin

**Status**: Accepted
**Date**: 2026-08-16
**Deciding ticket**: [`10` Wire the real S3 ObjectStore](../../.scratch/custodian/issues/10-real-s3-object-store.md)
**Amends**: [ADR-0001](0001-hosting-and-deployment-posture.md)

## Context

ADR-0001 settled that media lives in a private S3 bucket and that the edge is
"one CloudFront distribution fronting both persona and custodian". It did not
settle how uploaded media *bytes are read back* by the public. Two candidate
paths existed, and the codebase carried both intents at once:

- **Through the custodian origin.** `deed` ticket 06 and a comment in
  `deed/compute/buckets.tf` described routing `/media/<key>` to the box, so
  custodian (or nginx in front of it) would serve the bytes.
- **CDN-direct.** `CONTEXT.md` describes the media URL as a domain-owned CDN URL
  (`cdn.mihirsingh.dev/…`), implying the CDN serves the bytes from the bucket and
  custodian is not involved in the read.

custodian is the single, load-bearing box (ADR-0001): persona fetches blogs from
it on every page view, and it is a single point of failure. Putting media reads
through it adds bytes-serving load to the one process whose availability is the
site's primary content, and contradicts the reserve/confirm design where
custodian is deliberately never on the *write* byte path either.

## Decision

**Media is served CDN-direct.** The public reads media from a **dedicated
CloudFront distribution** — `cdn.mihirsingh.dev` — sitting in front of the
private S3 media bucket via Origin Access Control (OAC). custodian is **never on
the media byte path**, read or write:

- **Write**: broom `PUT`s to a custodian-presigned S3 URL. broom→S3.
- **Read**: browser→CDN→S3. custodian is not in the path.
- **custodian's role**: presign the upload, `HEAD` to confirm bytes landed,
  `DeleteObject` on delete, `HeadBucket` for the health gauge — and persist each
  media record's absolute CDN url so persona and log bodies reference the CDN.

### A separate distribution, not a behavior on the edge

`cdn.mihirsingh.dev` is a distinct hostname from custodian's API domain.
CloudFront routes to origins by path, not by Host header, so serving a second
hostname cleanly means a second distribution rather than a `/media/*` behavior
grafted onto the API distribution. This is the one place the playground runs two
CloudFront distributions, narrowing ADR-0001's "one distribution over both
components" to "one distribution for the API + persona, plus a dedicated media
CDN".

The cost is unaffected: CloudFront's free tier (1 TB egress, 10M requests/month)
is account-wide, not per-distribution, and the extra ACM certificate and Route 53
alias for `cdn.` are free.

## Consequences

### Good

- The load-bearing box never serves media bytes; media availability does not
  depend on custodian being up, and a media hotlink cannot add read load to the
  API process.
- The media URL is portable and CDN-native exactly as `CONTEXT.md` promises —
  custodian stores and hands back the absolute `cdn.mihirsingh.dev/<key>` url.
- The media bucket stays fully private; only the CDN reaches it, via OAC.

### Bad, or merely accepted

- Two CloudFront distributions instead of one — a second distribution, cert, and
  DNS alias to provision and reason about, against ADR-0001's single-distribution
  simplicity.
- `custodian` must be configured with the CDN base (`CUSTODIAN_MEDIA_CDN_BASE`);
  an unset base yields relative, unusable media urls. This is deploy config that
  must be supplied, not defaulted.

### Downstream constraints

- `deed` provisions the `cdn.` distribution, its us-east-1 ACM cert, the Route 53
  alias, the OAC, and the media bucket policy granting only that distribution
  `s3:GetObject`. Tracked as a `deed` ticket.
- `deed` ticket 06 no longer routes `/media/` to the custodian origin; media is
  off the API/persona distribution entirely.
