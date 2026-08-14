# 01b — Reconcile integration secrets to env-at-startup (`24`)

**Why this exists:** the walking skeleton (`01`, commit `6b04b34`) was built
against the pre-`24` decision, where integration credentials lived in SQLite and
were written through the admin API. Ticket `24` reverted that — integration
sources are code-defined adapters and their keys are environment secrets, the
same class as the admin token hash and OTLP token. The skeleton's scaffolding
therefore contradicts the updated custodian spec. It carries **no business
behaviour** (the credential table and endpoint are unused), so this is a clean
removal, not an unpick. Correct it before `06` builds the poller on top.

**What to fix (drift found reviewing `01` against the updated spec):**

- **Schema** — remove the `integration_credential` table (and its comment) from
  `custodian/internal/storage/migrations/0001_init.sql`. The `integration`
  timeseries table stays untouched. Since `01` is the only migration and nothing
  is deployed, edit the existing migration in place rather than adding a drop
  migration.
- **Contract** — remove `PUT /admin/v1/integrations/{source}/credential`
  (`putIntegrationCredential`) and the `IntegrationCredential` schema from
  `custodian/openapi/custodian.yaml`. Keep `GET /v1/integrations/{source}` and
  `POST /admin/v1/integrations/{source}/refresh`.
- **Generated code** — re-run `just gen` so `server.gen.go` and `client.gen.go`
  drop the removed operation; confirm `just gen-check` is clean in CI.
- **Config** — add the integration keys to the env-read set in
  `custodian/internal/config/config.go` (e.g. `CUSTODIAN_STEAM_KEY`,
  `CUSTODIAN_GITHUB_PAT`), alongside `AdminTokenHash` and `OTLPToken`, read via
  `os.Getenv`, source treated as opaque. The poller (`06`) will consume them.
- **Tests** — the skeleton harness asserts routing/auth/CORS/health only; if any
  test references the removed credential endpoint, drop it. No poller behaviour
  is in scope here — that stays `06`.

**Blocked by:** None — corrects already-shipped code; can start immediately.

**Status:** ready-for-agent

- [ ] `integration_credential` table removed from `0001_init.sql`; `integration` timeseries table unchanged
- [ ] Credential `PUT` path + `IntegrationCredential` schema removed from `custodian.yaml`; `refresh` and public read retained
- [ ] `just gen` re-run; `server.gen.go`/`client.gen.go` no longer expose the credential write; `just gen-check` clean
- [ ] Integration key env vars added to `config.go`, read via `os.Getenv`
- [ ] Any test referencing the removed endpoint dropped; harness still green
