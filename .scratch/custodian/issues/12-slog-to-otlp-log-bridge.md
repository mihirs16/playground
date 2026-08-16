# 12 — Bridge slog to OTLP so logs reach the monitoring plane

**What to build:** custodian's application logs land in the monitoring plane
(Grafana Cloud) over the same OTLP pipeline as metrics and traces, not just on
the box. Today `main.go` builds a stdout-only `slog` logger and the OTel
`LoggerProvider` in `otlp.go` is stood up but fed by nothing, so every
`logger.*` call goes to stdout → docker → journald and never exports — the logs
half of the observability decision (`11`, spec §Observability) was never
connected. This ticket wires custodian's existing `slog` calls through the OTel
log SDK (e.g. the `otelslog` bridge over the global `LoggerProvider`) while
keeping stdout intact, so `journald`/`docker logs` stays the short-retention
local fallback the spec calls for and Grafana becomes the durable home. No new
log call sites and no new log statements — this only changes where the logs
custodian already emits are delivered.

**Begin with cost + retention, before any code:** the first work item is to
**estimate the cost** of shipping logs to Grafana Cloud (log volume at steady
state and under an incident burst → Loki ingestion/retention against the free
tier and the ~£15/mo envelope; note the poll loop logs on a 60s cadence now) and
to **make the log-retention policy explicit** — how long logs are kept in
Grafana, how long the local `journald` fallback buffers, and what (if anything)
gets sampled or dropped to stay in budget. Record both in the ticket / an ADR
before wiring the bridge, because retention and sampling decisions shape how the
bridge is configured (severity floor, batching, attribute set). If the estimate
shows logs would breach the envelope, surface that and stop for a decision rather
than silently shipping everything.

**Blocked by:** 07 (OTLP export + LoggerProvider already stood up). Independent
of the OTLP auth fix and of 10/11.

**Status:** ready-for-agent

- [ ] Log-volume + Loki cost estimate written down (steady-state + burst) against the ~£15/mo envelope
- [ ] Retention policy made explicit: Grafana retention, local `journald` fallback window, any sampling/severity floor
- [ ] custodian's existing `slog` calls export over the OTel `LoggerProvider`; no new log call sites added
- [ ] stdout logging retained so `docker logs` / `journald` still works as the local fallback
- [ ] An empty/misconfigured OTLP endpoint degrades to stdout-only cleanly (no boot failure, matching the metrics path)
- [ ] Logs for a known event (startup, a poll warning) are visible in Grafana Cloud once auth is configured
