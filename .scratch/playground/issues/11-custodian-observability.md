# custodian: observability, monitoring and alerting

Type: grilling
Status: open
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

- A **single hand-administered EC2 `t4g.micro`** in eu-west-1. No managed APM arrives by default — whatever exists, you install and run.
- **CloudWatch is in reach** and coherent with everything else being AWS: logs, metrics, alarms, and SNS to a channel that reaches you. Worth weighing against Grafana/Prometheus on the box (more learning, more to maintain, and it dies when the box dies) and against hosted free tiers.
- **A single instance is a single point of failure**, which raises the value of an *external* prober. A health check running on the box cannot tell you the box is gone.
- **CloudFront sits in front of custodian**, so edge-cached blog responses may keep serving briefly after the origin dies. That softens the "custodian is down" question in the body — but it also means a naive uptime check hitting the public URL can pass while the origin is dead. Probe the origin, not just the edge.
- **1 GB of RAM** is a real budget. An agent-heavy observability stack is not free here.

Still blocked on `07`, since instrumentation libraries are language-specific.

The last question above is the sharpest one and connects to `16`. Runtime-loaded blogs mean custodian's availability is the site's availability for its primary content. That's a deliberate trade already made — this ticket decides how to make it survivable rather than whether to accept it.
