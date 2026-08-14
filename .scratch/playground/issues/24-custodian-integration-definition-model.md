# custodian: integration definition model

Type: grilling
Status: resolved
Blocked by: 03, 12, 19

## Question

Are integration sources (Steam, GitHub, future) defined in custodian's code, or registered dynamically via broom — and where do their credentials live as a consequence?

`19` split integration secrets into their own class (SQLite, broom-written, runtime-read) on the premise that integrations are an "operational, growing set" configured without a redeploy. That premise only holds if integrations are *dynamically definable*. So the definition model and the secret location are one coupled decision:

- **Fully in code** — each source is a hand-written adapter (endpoint, auth, response-shaping) compiled into custodian. The set is fixed at build time; a new source is a code change + redeploy.
- **Fully generic** — broom registers a new source end-to-end (URL, auth, cadence, field extraction) with zero custodian code; custodian ships a config-driven generic poller.

No halfway house (a code-backed registry where broom enables/credentials known adapter *types*) — the owner ruled that out explicitly.

## Context

Surfaced while orienting on the map: the owner wanted integrations "set up via broom, not hardcoded," which `19` had partially delivered (credentials broom-managed) while leaving adapters in code. Pulling *definition* into broom would have redrawn scope — `03` ruled the generic user-defined-source idea a future direction, not v1.

The deciding asymmetry: generic *fetching* is cheap, but generic *shaping* is not — Steam's recently-played is a nested 2-week aggregate and GitHub's events feed needs filtering + ETag conditional requests + reshaping into the predictable per-widget shape persona renders. A generic model must push that mapping either into broom config (a field-extraction mini-language, leaking into `08`/`09`) or into persona (making it source-aware, against `12`'s "custodian owns derived shaping").

## Answer

**Fully in code.** Integration sources are hand-written adapters compiled into custodian; the set is fixed at build time and a new source is a deliberate code change + redeploy. The generic-source model was weighed and dropped — for a two-source personal site where a third source is a rare event, the config-driven-shaping machinery would be built and never amortised, and it fights `12`'s custodian-owns-shaping line. The generic model stays fog with the same "capable-of, don't-build" framing as `03` (unchanged).

**Credentials are startup env secrets, not broom-managed.** Because the adapter set is fixed and known at build time, so is the credential set — which puts it exactly in `10`/`19`'s **bootstrap/identity** secret class (SSM SecureString → tmpfs env → `os.Getenv`), alongside the admin-token hash and the Grafana OTLP credential. There is no second "operational secret" class and no integration secret rows in SQLite.

**This REVISES `19`:** its "two secret classes" collapses to **one — env-at-startup**. The "integration secrets live in custodian's SQLite, broom-written, runtime-read per poll" half is removed; the self-hosted-secrets-manager rejection in `19` still stands (it was never the chosen path). `deed` provisions the SSM params + read grant identically for all bootstrap secrets. `10` is **not** revised — it already specified env-at-startup; `19` had been the thing that moved integration keys env→DB, and that move is now undone.

**Integrations drop off broom's surface entirely for now.** With credentials in env and definitions in code, broom has no integration role left. **REVISES `17`:** the `broom integration refresh [name]` verb and the `integration` noun-group are removed from broom's command surface. Custodian's `POST /admin/v1/integrations/{source}/refresh` endpoint (`09`/`12`) is unaffected — it remains a curl-able admin/debug endpoint, just no longer wrapped by broom.

**Unaffected:** `03`'s `integration` *record* (cached derived data + fetch timestamp, id+body shape) — that's fetched data, never the key. `12`'s poller, cadence, idle model, and append-on-change storage all stand. The generic-source / user-defined-type direction stays out of v1 (`03`).
