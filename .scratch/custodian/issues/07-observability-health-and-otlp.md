# 07 — Observability: health gauge, OTLP export & admin access log

**What to build:** custodian becomes observable and self-reports a single clear
health signal that doubles as its heartbeat. On a timer (reusing the existing
poll loop), custodian self-assesses health — `SELECT 1` on SQLite, an S3
`HeadBucket`, and disk/memory headroom — and emits a single `health` gauge (`1`
healthy / `0` degraded); third-party API reachability is excluded entirely, so
Steam or GitHub being down never turns custodian red. Instrumentation uses the
OTel Go SDK, exporting metrics, traces, and logs over OTLP/HTTP directly to a
configured endpoint (endpoint URL + export token read from the environment at
startup, source opaque) — no agent sidecar. Every `/admin/*` access is logged
(timestamp, real client IP from CloudFront's `X-Forwarded-For`, path, result) so
any admin call the author did not make is a visible leak signal. A trivial
debug-only `/healthz` endpoint exists for manual `curl` but is not a
load-bearing public contract. The OTLP exporter is exercised through its fake /
no-op sink so tests can assert the gauge value is computed and emitted without a
live backend.

**Blocked by:** 06.

**Status:** done

- [x] `health` gauge computed on a timer from `SELECT 1` + S3 `HeadBucket` + disk/mem headroom, emitting `1`/`0`
- [x] Third-party API reachability excluded from health (fake third-party failure does not flip the gauge)
- [x] OTel Go SDK exports metrics + traces + logs over OTLP/HTTP to a config-URL endpoint; endpoint + token from env
- [x] `/admin/*` access logged with timestamp, real client IP (via XFF), path, result
- [x] Debug-only `/healthz` responds for manual curl
- [x] Gauge value asserted through the fake OTLP sink without a live backend
