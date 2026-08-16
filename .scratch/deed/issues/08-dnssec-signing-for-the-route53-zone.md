# 08 — DNSSEC signing for the Route 53 zone

**What to build:** Cryptographically sign the `mihirsingh.dev` zone so validating
resolvers can prove an answer genuinely came from the zone and was not forged in
transit. `deed` creates the key-signing key, enables signing on the hosted zone,
and the DS record that anchors the chain of trust is published at the registrar
(Squarespace). The zone is unsigned today (Route 53 does not sign by default); this
closes that gap.

**Blocked by:** 05 (the hosted zone must exist and be delegated to Route 53).

**Status:** ready-for-agent

- [ ] KMS asymmetric signing key (`ECC_NIST_P256`, key usage `SIGN_VERIFY`) created **in us-east-1** — Route 53 requires the key-signing key there regardless of where the zone lives. Its key policy must let the Route 53 DNSSEC service (`dnssec-route53.amazonaws.com`) use it to sign.
- [ ] `aws_route53_key_signing_key` + `aws_route53_hosted_zone_dnssec` enable signing on the zone; the zone publishes `DNSKEY`/`RRSIG` records.
- [ ] DS record published at Squarespace matching the KSK, in the correct order — **sign the zone first, then add the DS** (adding the DS while the zone is unsigned SERVFAILs the whole domain). Verified: a validating lookup returns the AD (authenticated-data) bit and no SERVFAIL, and a DNSSEC analyzer shows a complete chain from the `.dev` registry down.

## Notes & risks

- **Registrar capability is a precondition to confirm first.** `.dev` is a signed
  TLD (Google's registry), so DS records are supported — but the *registrar UI*
  (Squarespace, post-Google-Domains) must expose a way to enter a custom DS record.
  If it does not, this ticket is blocked on registrar support and drops to
  `ready-for-human` / `wontfix` rather than being worked around.
- **Ordering is the whole game.** Enable: zone-signing on **before** the DS at the
  registrar. Disable/rollback: remove the DS at the registrar **first**, wait out
  its TTL, then turn signing off. Getting this backwards takes the domain (and
  email) dark until caches expire.
- **Placement.** The KSK and signing resources attach to `aws_route53_zone.playground`,
  which currently lives in `deed/compute`. If the zone is later extracted into its
  own `deed` component, these move with it.
- **This is a hardening feature, not a fix.** The zone is fully functional unsigned;
  most domains run without DNSSEC. Priority is low and independent of the custodian
  deploy workstream.
