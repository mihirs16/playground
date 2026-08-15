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

**Status:** ready-for-agent

- [ ] Single CloudFront distribution provisioned with the custodian box as API origin
- [ ] Public TLS certificate issued via ACM; no certificate private key material on disk
- [ ] DNS delegated to Route 53 with edge + origin records; custodian reachable same-origin over HTTPS at its domain
