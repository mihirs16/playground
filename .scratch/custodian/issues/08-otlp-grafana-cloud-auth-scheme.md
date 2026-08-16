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

**Status:** done

Notes:
- `CUSTODIAN_OTLP_ENDPOINT` is the *base* OTLP/HTTP endpoint (the standard
  `OTEL_EXPORTER_OTLP_ENDPOINT` convention); custodian appends `/v1/<signal>`
  itself, since the SDK's `WithEndpointURL` takes the path verbatim. For Grafana
  Cloud that means the `.../otlp` gateway is hit at `.../otlp/v1/metrics`.
- Telemetry lands as `service.name=custodian`, `service.namespace=playground`.
- Only the `health` gauge is emitted today; the trace/log providers are wired
  but nothing feeds them yet (custodian's slog is not bridged to the OTel log
  provider, and no spans are created), so "logs/traces" arrive empty for now.

- [x] custodian authenticates to Grafana Cloud's OTLP gateway with the scheme it expects (Basic `instanceID:token`), replacing the hardcoded `Bearer` — via scheme-agnostic verbatim `Authorization` header from `CUSTODIAN_OTLP_AUTHORIZATION`
- [x] A misconfigured/misauthenticated exporter is distinguishable from "no endpoint configured" — the silent no-op fallback no longer hides an auth failure — build failure returns an error (logged in `real.go`); runtime 401s route through custodian's logger via the OTel error handler
- [x] Empty-endpoint → no-op behaviour is preserved
- [x] The env-var contract is reconciled with `deed`'s `otlp-credential` bootstrap secret and documented
- [x] **Deliverable — local verification against a real stack:** ran the built binary locally against `https://otlp-gateway-prod-gb-south-1.grafana.net/otlp` with the real `Basic <instanceID:token>` credential; the periodic metric export completed with no `OTLP export failed` from the error handler (i.e. the gateway returned 2xx), confirming auth + path reach Grafana Cloud end-to-end. Visual confirmation in the Grafana Cloud UI is the operator's to eyeball (`service.name=custodian`, `service.namespace=playground`, metric `health`).
