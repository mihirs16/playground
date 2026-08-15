# custodian: observability, monitoring and alerting

Type: grilling
Status: resolved
Blocked by: 07

## Question

How do you find out custodian is unhealthy, and what counts as unhealthy?

- **Health endpoint** — what does it actually check? Liveness only, or readiness including the store and object storage? Does third-party API reachability count, given Steam being down shouldn't mean custodian is down?
- **Alerting** — what channel reaches you, and what threshold fires? A personal project needs alerts you'll actually see and won't learn to ignore.
- **Uptime checking** — external prober (Better Stack, Healthchecks.io, UptimeRobot) or host-native? External is more honest, since it catches the host itself failing.
- **Logging** — structured or plain, retained where, for how long?
- **Metrics and tracing** — are they worth it at this scale, or is that resume-driven complexity? The playground exists partly to learn, so "worth doing because I want to learn it" is a legitimate answer here — just make it an explicit one.
- **Degradation, not just failure** — if Steam is unreachable, does the status widget disappear, show stale data with a timestamp, or show an error? Stale-with-timestamp is usually kindest.
- **What if custodian is down entirely?** Blogs load at runtime, so persona shows an empty blog list. Is there a fallback — a stale cache at the edge, a build-time snapshot as backstop — or is that accepted?

## Context

You raised monitoring and alerting yourself when deciding that blogs load from custodian at runtime, on the reasoning that custodian would need health monitoring anyway. This ticket exists because that made it a stated requirement rather than an assumption.

**Unblocked by `01`.** What it settled, and what it means here:

- A **single hand-administered EC2 `t4g.micro`** in eu-west-2. No managed APM arrives by default — whatever exists, you install and run.
- **CloudWatch is in reach** and coherent with everything else being AWS: logs, metrics, alarms, and SNS to a channel that reaches you. Worth weighing against Grafana/Prometheus on the box (more learning, more to maintain, and it dies when the box dies) and against hosted free tiers.
- **A single instance is a single point of failure**, which raises the value of an *external* prober. A health check running on the box cannot tell you the box is gone.
- **CloudFront sits in front of custodian**, so edge-cached blog responses may keep serving briefly after the origin dies. That softens the "custodian is down" question in the body — but it also means a naive uptime check hitting the public URL can pass while the origin is dead. Probe the origin, not just the edge.
- **1 GB of RAM** is a real budget. An agent-heavy observability stack is not free here.

Still blocked on `07`, since instrumentation libraries are language-specific.

The last question above is the sharpest one and connects to `16`. Runtime-loaded blogs mean custodian's availability is the site's availability for its primary content. That's a deliberate trade already made — this ticket decides how to make it survivable rather than whether to accept it.

## Answer

**One monitoring plane — Grafana Cloud free tier — with custodian instrumented agentlessly via OpenTelemetry. No CloudWatch, no on-box Prometheus/Grafana, no separate uptime prober, no blank status page in v1.** The design centralises everything operational in Grafana Cloud (dashboards + mobile app), and leans on the map's "self-manage compute, rent durability" principle by renting a *fixed free tier* rather than a metered service.

### What "unhealthy" means — health logic, endpoint now debug-only

- Custodian **self-assesses** on a timer: `SELECT 1` against SQLite, a cheap S3 reachability check (`HeadBucket`), and **disk/mem headroom** (self-collected, e.g. `gopsutil`). This produces a single **`health` gauge: `1` healthy / `0` degraded**.
- **Third-party APIs (Steam/GitHub) are excluded from health entirely.** Their reachability is the *poller's* concern and surfaces as stale derived data, never as custodian being unhealthy. Steam being down must not turn custodian red.
- The self-collected host stats feed **two consumers**: the `health` gauge (pushed to Grafana) and the degraded verdict.
- **HTTP health endpoint is now debug-only.** With the status page dropped (below), no external consumer pulls health — Grafana is push-fed. Keep a trivial `/healthz` for manual `curl`/future use since it's near-free, but it is **not a load-bearing public contract**. (The richer `/readyz` HTTP surface envisaged earlier is unnecessary once nothing consumes it over HTTP; the *logic* lives on regardless.)

### Instrumentation — agentless OpenTelemetry (metrics + traces + logs)

- Custodian embeds the **OTel Go SDK** and exports **OTLP/HTTP directly** to Grafana Cloud's native OTLP endpoint (basic auth: instance ID + token). **No Grafana Alloy / agent sidecar** — nothing extra to run on the 1 GB `t4g.micro`, single-binary ethos preserved.
- **Vendor-neutral by construction:** the backend is a config value (endpoint URL). Grafana Cloud today; Honeycomb/Dash0/self-hosted collector later without touching instrumentation. Mirrors the `blank`/`deed` "don't get shaped by one consumer" ethos, and OTel is the more transferable thing to learn than a Prometheus-specific pipeline.
- **All three signals via OTLP: metrics, traces, and logs.** Logs go through OTLP → Loki despite the OTel Go logs bridge being newer — judged fine at this project's scale. On-box `journald` kept only as a **short-retention local fallback buffer** for when egress/Grafana is unavailable; not the primary store.
- **`/metrics` learning goal is satisfied by OTel**, not a hosted Prometheus server (which would die with the box and point the wrong way). "Worth doing to learn OTel" is the explicit, blessed motivation.

### Uptime + alerting — heartbeat-as-metric, one rule, mobile push

- The heartbeat is **not a separate dead-man's-switch service** — it is the `health` gauge itself. Custodian emits it continuously; **absence of data over a window *is* the missed-heartbeat signal.**
- **One Grafana alert rule** covers everything: fire on **`health == 0` OR `no data for <window>`** (a few missed intervals → grace window kills flapping). Degraded and down are the same rule.
- **Contact point → Grafana IRM → Grafana mobile app push.** Confirmed available on the **free tier** (3 IRM users; mobile push is the recommended primary notification method). Note: self-hosted Grafana OnCall OSS is being archived (~2026) → folded into Grafana Cloud IRM, but we're on Cloud, so we're on the supported path. Chosen over email (inbox noise) and Discord (flaky mobile delivery).
- **Accepted single-vendor caveat:** if Grafana Cloud/OTLP ingestion is itself down, a "no data" rule can false-positive and Grafana can't watch itself. Acceptable at this scale; it's the price of the centralisation.

### Status page — dropped for v1

- The blank status page is **dropped for now** — Grafana's dashboards + mobile app *are* the monitoring surface, and building a second one duplicated effort for a v1 whose real goal is learning OTel. Reading a public blank page from Grafana would also need public dashboards or a server-side token proxy (a naked client-side Grafana token would repeat the `NEXT_PUBLIC_NOTION_KEY` mistake), which isn't worth it now.
- Folds into **fog** under persona's page/route inventory — revivable later. If revived, the honest live-status source is custodian's own `/healthz` (token-free, fails-closed when the box is down), with historical uptime coming from Grafana via a server-side proxy.

### Degradation vs. failure — custodian serves last-known, persona decides

- **Derived-data widgets keep showing the last-known value** when Steam/GitHub is unreachable. **No max-age cutoff in custodian, no mandated staleness display.** Custodian's contract is "here's the last good value and its `updated_at`" (metadata already in the `08` schema); **persona** reads `updated_at` and decides entirely how/whether to signal age. Hands a clean fact to `12` (freshness) and `16` (persona).

### Custodian down entirely — accept + edge-cache, snapshot deferred

- **Posture (a)+(b):** runtime-load blogs from custodian; **CloudFront edge cache** (`01` fronting + `09`'s `Cache-Control`/`ETag` read surface) softens brief outages by serving stale cached blog responses for their TTL; **accept an empty / "temporarily unavailable" blog state** for a genuine extended outage.
- **Build-time snapshot backstop (c) deferred to fog.** Baking a blog snapshot at build time quietly reintroduces the build-to-edit staleness custodian exists to escape, and adds real persona complexity. The mitigation for "custodian down = blogs down" is **keeping custodian up** (heartbeat + alerting), not papering over its absence. `(c)` returns only if uptime proves inadequate.

### Cost

Observability adds **~£0/month**: Grafana Cloud free tier (no payment method attached = hard off-switch, the structural answer to the CloudWatch budget-leak worry), agentless (no extra compute), logs/metrics/traces within free allotments by low-cardinality discipline. Does **not** move the `01` budget.
