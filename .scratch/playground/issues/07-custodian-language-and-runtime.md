# custodian: language and runtime

Type: grilling
Status: resolved

## Question

What language and runtime is custodian written in?

Judge against what custodian actually has to do: serve public reads to persona at runtime (including blogs, so it's on the critical path for page content), accept authenticated writes and multipart media uploads from the cli, poll and cache two third-party APIs, and stay up with monitoring and alerting.

Candidates worth weighing rather than a full survey:

- **Go** — single static binary, excellent stdlib HTTP, trivially cheap to host, strong operational story. The obvious "boring API" choice.
- **TypeScript/Node** — shares types with `blank` and persona, one language across most of the repo, largest ecosystem for markdown tooling. Weakest of the three operationally.
- **Rust** — most learning value if that's wanted, best resource profile, slowest to write and the heaviest maintenance burden for a solo side project.

## Context

**Unblocked by `01`, which hands down three hard constraints:**

- Must run comfortably in **1 GB of RAM on ARM64** (EC2 `t4g.micro`, Graviton). Resizing to 2 GB is a two-minute stop/start, so this is a preference rather than a wall — but a runtime that *needs* the resize on day one should say so out loud.
- Must **embed SQLite in-process**. State is a file on the instance's disk, continuously replicated to S3.
- Must have an **AWS SDK supporting the instance-metadata credential chain**, since custodian holds no long-lived credentials.

A **long-lived process** is assumed — serverless is off the table. Every candidate below satisfies all three constraints, so this narrows the field less than it might look.

What `01` *did* change is the weighting. The box is hand-administered — nginx, systemd, TLS, deploys, and whatever `11` decides about monitoring, all owned personally. "Boring and debuggable at 11pm" is worth more than it was when this ticket was written.

Standing constraint from charting: **polyglot is incidental, not a goal** — do not pick a language here for variety's sake. If TypeScript is genuinely the best tool, two-thirds of the repo being TypeScript is a fine outcome.

Counter-pressure worth taking seriously: custodian is the component that has to *keep running* for the site to render blogs. A language chosen for novelty and later resented is the most likely way this project dies. Weight "will I still want to maintain this in two years" heavily.

Type-sharing with persona is a real but overrated argument — a generated OpenAPI client gives most of the benefit across a language boundary, and `09` will settle the contract format anyway.

## Blocks

`11` observability, `13` monorepo tooling, `17` cli language

## Answer

**Go**, as a single static binary.

- **Language: Go.** The boring, debuggable-at-11pm choice for the one component that has to *stay up* for the site to render blogs — and, decisively, **career-aligned learning** for the author, so the "will I still want to maintain this in two years" risk and the "learn something" motivation point the *same* way. No maintenance-vs-novelty trade to make. (Rust was the novelty option, rejected on maintenance burden; TS was the one-language-repo option, rejected because its only real edge — markdown tooling and type-sharing — is weak here: goldmark is a GFM-compliant Go parser, and type-sharing goes through the OpenAPI client `09` will define.)
- **SQLite driver: pure-Go `modernc.org/sqlite`.** `CGO_ENABLED=0`, so the ARM64 binary cross-compiles straight from the author's laptop with no C toolchain — the single-static-binary operational story survives intact. The modest perf gap vs cgo `mattn/go-sqlite3` is irrelevant at personal-site traffic. S3 replication (Litestream-style) is file-level and driver-agnostic, so this choice doesn't constrain `08`.
- **HTTP stack: chi.** A `chi.Router` *is* an `http.Handler`, so there's no framework lock-in and no rewrite ceiling if custodian later fronts other projects — it scales via sub-routers and versioned route groups, with clean composable middleware for auth/logging/recovery. Chosen over stdlib-only (chi gives the middleware ergonomics without hand-rolling) and over Gin/Echo (which couple every handler to a framework context) / Fiber (not `net/http`-based).
- **Settled facts, not open decisions:** AWS SDK for **Go v2** covers the IMDS credential chain natively (satisfies the "no long-lived credentials" constraint from `01`); **Go 1.22+** `ServeMux` underpins chi's routing; **goldmark** is available if custodian ever needs server-side GFM.
- **Deploy shape** unchanged from `01`: one static binary under systemd, behind nginx. Nothing new decided here.

Comfortably inside the 1 GB / ARM64 envelope — a static Go binary with an embedded pure-Go SQLite is one of the lightest-footprint options on the table, so no day-one resize to 2 GB.
