# 08 — OTLP export auth: reconcile Bearer with Grafana Cloud's Basic scheme

**What to build:** custodian's OTLP exporter must actually authenticate against
the intended backend. Today it hardcodes a single scheme — `edges/otlp.go` sets
`Authorization: Bearer <token>` (`CUSTODIAN_OTLP_TOKEN`) on all three exporters.
Grafana Cloud's OTLP gateway expects HTTP **Basic** auth, where the credential is
`base64("<instanceID>:<token>")` sent as `Authorization: Basic …`. As written,
custodian's telemetry will be rejected (401) by Grafana Cloud even with a valid
token, and the failure is silent: a build error in the exporter falls back to the
no-op sink (`otlp.go:41`), so a misauthenticated export looks identical to "no
telemetry configured" — nothing surfaces at startup.

Resolve the mismatch so a real Grafana Cloud stack receives custodian's metrics,
traces, and logs. Pick the approach in the ticket discussion — the two obvious
shapes are (a) make custodian send the full `Authorization` header value
verbatim from a single env var (the deploy wrapper assembles `Basic …`, keeping
custodian scheme-agnostic and matching the config package's "source is opaque"
ethos), or (b) accept instanceID + token and have custodian build the Basic
header. Keep the empty-endpoint → no-op behaviour, but a *misconfigured* export
(e.g. a build/auth failure) should be distinguishable from "telemetry off" rather
than silently swallowed.

This also reconciles the bootstrap-secret contract in `deed`
(`deed/compute/terraform.tfvars.example`, `otlp-credential`): whatever env var
custodian settles on is what the deploy wrapper injects from SSM.

**Blocked by:** 07.

**Status:** ready-for-agent

- [ ] custodian authenticates to Grafana Cloud's OTLP gateway with the scheme it expects (Basic `instanceID:token`), replacing the hardcoded `Bearer`
- [ ] A misconfigured/misauthenticated exporter is distinguishable from "no endpoint configured" — the silent no-op fallback no longer hides an auth failure
- [ ] Empty-endpoint → no-op behaviour is preserved
- [ ] The env-var contract is reconciled with `deed`'s `otlp-credential` bootstrap secret and documented
- [ ] **Deliverable — local verification against a real stack:** run custodian locally pointed at an actual Grafana Cloud OTLP endpoint with a real token, and confirm the `health` gauge (and logs/traces) arrive in Grafana Cloud — i.e. the export is observed end-to-end, not just asserted through the fake sink
