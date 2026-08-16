# 01 — Walking skeleton: server, storage seam, codegen & test harness

**What to build:** A booting custodian binary that stands up the whole shape the
rest of the work hangs off, with nothing business-facing yet. A chi server
exposes two sub-routers — public `/v1/*` and admin `/admin/v1/*` — as one
`http.Handler`. It opens a real in-process `modernc.org/sqlite` database
(`CGO_ENABLED=0`) and runs migrations at startup. The three outward edges that
leave the box — S3 (presign / `HEAD` / bucket ops), the Steam/GitHub HTTP
clients, and the OTLP sink — are defined as interfaces injected at construction,
each with a fake for tests and a real implementation stubbed. The hand-authored
OpenAPI 3.1 `.yaml` lives here as the source of truth, with `just gen` wiring
that fans server interfaces (oapi-codegen) and a client out per `13`. A
`net/http/httptest` black-box harness stands the real router up against a real
temp/`:memory:` sqlite and the fake edges. Config (including the CORS allowlist
and per-source poll intervals) is read from the environment via `os.Getenv`,
source treated as opaque.

**Blocked by:** None — can start immediately.

**Status:** done

- [ ] `go build` produces a single static ARM64-cross-compilable binary (`CGO_ENABLED=0`)
- [ ] Server boots, runs migrations against `modernc.org/sqlite`, and serves both sub-routers
- [ ] S3, third-party HTTP client, and OTLP sink each exist as an injected interface with a working fake
- [ ] OpenAPI 3.1 `.yaml` skeleton committed; `just gen` produces server interfaces + a vendored client and is CI-drift-checked
- [ ] `httptest` harness boots the real chi router + real sqlite + fake edges and issues a real request
- [ ] Config (CORS allowlist, poll intervals, secret env vars) read via `os.Getenv` only
