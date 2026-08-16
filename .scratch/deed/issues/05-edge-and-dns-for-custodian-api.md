# 05 — Edge & DNS for the custodian API: CloudFront + ACM + Route 53

**What to build:** Custodian served at its real domain over HTTPS through the
edge, same-origin per ADR-0001. `deed` provisions the single CloudFront
distribution with the custodian box as the API origin, the distribution's public
TLS certificate issued through ACM (AWS holds the private key — no cert material
on disk), and DNS delegated to Route 53 (registration stays at Squarespace) with
the edge and origin records declared alongside the rest of the shell. This is the
one distribution the whole playground fronts; persona's private origin and its
routing land in 06.

**Blocked by:** 02.

**Status:** done

- [x] Single CloudFront distribution provisioned with the custodian box as API origin — `deed/compute/edge.tf` (`aws_cloudfront_distribution.edge`: one custom origin `custodian` → `origin.<custodian_domain_name>`, default behavior forwards all-viewer-except-Host with caching disabled so custodian's auth/cookies work; `PriceClass_100`). The box gets a stable `aws_eip.box` and its security group now allows HTTP ingress from the `com.amazonaws.global.cloudfront.origin-facing` prefix list only (`deed/compute/main.tf`)
- [x] Public TLS certificate issued via ACM; no certificate private key material on disk — `deed/compute/edge.tf` (`aws_acm_certificate.edge` for `custodian_domain_name` via the `aws.us_east_1` aliased provider in `deed/compute/providers.tf`, DNS-validated through `aws_acm_certificate_validation.edge`; AWS holds the private key, nothing lands on disk)
- [x] DNS delegated to Route 53 with edge + origin records; custodian reachable over HTTPS at its domain — `deed/compute/edge.tf` (`aws_route53_zone.playground` for the apex `zone_name` is the delegation target — name servers surfaced via the `route53_name_servers` output for Squarespace; edge A/AAAA aliases point `custodian_domain_name` at the distribution, `origin` A record points at the EIP; viewer protocol policy `redirect-to-https`)

Implemented and `terraform fmt`/`validate` pass on the `deed` branch. custodian is
served at `custodian_domain_name` (`custodian.mihirsingh.dev`); the apex `zone_name`
(`mihirsingh.dev`) is persona's front-facing website and its routing on this same
distribution lands in 06.

**Human-verified:** applied to account `136102212434` / `eu-west-2` (cert in
`us-east-1`) under SSO credentials. The distribution (`E2VPVSL1RV9JCB`), the
us-east-1 ACM certificate (issued after DNS validation), the Elastic IP origin, and
the Route 53 zone are live. Domain delegated: Squarespace name servers switched to
the zone's four `awsdns` servers, DNSSEC left disabled (Route 53 does not sign by
default; enabling the registrar DS record without zone signing would SERVFAIL the
domain). End-to-end: `https://custodian.mihirsingh.dev` returns `504` from CloudFront
(`Via`/`X-Amz-Cf-Pop: LHR5`) over a valid TLS handshake — the edge, cert, and DNS are
proven; the `504` is the origin box having no custodian running yet, which the deploy
workstream (`deed/07` + `custodian/09`) resolves.

**DNS parity note:** delegating moved the whole zone, so `deed` also carries every
pre-existing record for the domain (`deed/compute/external-dns.tf`): Mailgun email
(MX + SPF + DKIM), the apex + `www` Netlify site, the `monteapi`/`projects` services,
and the Google verification CNAME. All were verified resolving from Route 53 before
the name-server switch, so email and the other services survived the cutover. The one
record intentionally dropped was `_domainconnect` (a Google-Domains-only helper, dead
once off Google's DNS).
