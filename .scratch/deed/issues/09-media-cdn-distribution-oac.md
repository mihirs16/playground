# 09 — Media CDN: dedicated cdn.<domain> distribution, OAC over the private media bucket

**What to build:** the public read path for uploaded media, served CDN-direct
from the private S3 media bucket and never through the custodian origin
(ADR-0002). `deed` provisions a **second, dedicated CloudFront distribution**
aliased to `cdn.<domain>` (distinct from the custodian/persona edge from 05/06,
because CloudFront routes to origins by path, not by Host — a separate hostname
wants its own distribution), fronting the `aws_s3_bucket.media` bucket via Origin
Access Control. The media bucket stays fully private: a bucket policy grants
`s3:GetObject` only to this distribution (scoped by `aws:SourceArn`), which is
compatible with the existing public-access block since an OAC policy is not a
public policy. A us-east-1 ACM certificate covers `cdn.<domain>`, and a Route 53
alias points the subdomain at the distribution. The default behavior serves
objects by key (`cdn.<domain>/<key>`), matching the extension-free, log-unscoped
url custodian records and hands back.

The distribution caches: media is immutable-by-key (custodian never overwrites a
reserved key), so a long TTL is safe and a delete is a bucket-object removal, not
an invalidation.

**Blocked by:** 04 (media bucket), 05 (zone + us-east-1 ACM provider wiring).

**Status:** ready-for-agent

- [ ] Dedicated CloudFront distribution aliased to `cdn.<domain>`, separate from the custodian/persona edge
- [ ] Distribution fronts the private media bucket via OAC; bucket policy grants `s3:GetObject` only to this distribution
- [ ] Media bucket stays private — existing public-access block unchanged, no public-website endpoint
- [ ] us-east-1 ACM cert for `cdn.<domain>`, DNS-validated through the zone
- [ ] Route 53 alias points `cdn.<domain>` at the distribution
- [ ] `cdn.<domain>/<key>` serves the object bytes with a cache-friendly TTL
- [ ] `CUSTODIAN_MEDIA_CDN_BASE` documented/wired to `https://cdn.<domain>` so custodian records absolute CDN urls
