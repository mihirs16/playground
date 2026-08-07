# custodian: auth model

Type: grilling
Status: resolved
Blocked by: 09

## Question

How does the cli authenticate to custodian, and what exactly is public?

- **Write auth** — a long-lived personal token, an OAuth device flow, mTLS, or something else? There is exactly one human writer, so the simplest thing that is genuinely safe probably wins.
- **Token storage on the client** — where does the cli keep credentials? OS keychain, a config file with restricted permissions, or an environment variable? What does `login` / `logout` look like, if those exist at all?
- **Is the read path fully public?** Blogs and profile presumably yes. Drafts presumably not — which means at least two trust levels, and a preview mechanism if persona is to render unpublished work.
- **Rotation and revocation** — what happens when a token leaks, and how would you know?
- **Rate limiting and abuse** — the read API is public and on the critical path for page rendering. What stops it being hammered?
- **Third-party secrets** — where do the Steam and GitHub API keys live, and who can reach them? These are custodian-side only and must never be exposed to persona's client bundle.

## Context

Blocked on `09` because auth attaches to a contract — whether there's one API or two changes the answer substantially.

**The cautionary fact this ticket exists to avoid repeating**: the deprecated site set `NEXT_PUBLIC_NOTION_KEY`, and the `NEXT_PUBLIC_` prefix inlines a value into the client bundle. The Notion integration token was therefore shipped to every visitor's browser. The same failure mode is available here — persona fetches from custodian at runtime, in the browser, so anything persona needs to send is public by construction. That constraint should shape the design rather than be patched later.

Corollary worth stating explicitly in the spec: because persona's Steam and GitHub reads happen client-side, custodian must proxy them. The third-party keys stay server-side, always.

## Answer

Six branches, grilled one at a time. Auth guards only `/admin/*` (from `09`) and the credential rides a header, never a cookie (also `09`).

### 1. Write-auth mechanism — hashed bearer token

A single long-lived **bearer token** on `Authorization: Bearer <token>`; custodian stores only a **hash**, so a DB/S3 leak yields nothing usable. No OAuth device flow, no mTLS, no login server — those solve multi-user / delegated-access / browser-session problems that a single writer on a single box does not have. mTLS was weighed (credential never appears in a loggable header) and dropped as ceremony against the "boring, one writer" grain. **`for now` flagged** — the upgrade path (mTLS, or scoped tokens) lives in fog, not foreclosed.

### 2. Client token storage — config file, no keychain

`broom` keeps the token in a **`0600` config file** under `~/.config/broom/` (XDG-aware), with a **`BROOM_TOKEN` env override that wins when set** (for CI / keeping it off disk). `broom login` prompts/reads the token, writes it `0600`, and **verifies it with one authenticated call** (e.g. `GET /admin/v1/logs`) so a bad paste fails immediately; `broom logout` deletes the file. OS keychain (Keychain/libsecret) deliberately skipped — it defends against other local users reading your dotfiles, which is not this machine's threat model, at the cost of a platform-specific dependency.

### 3. What is public — two log states, no private third state

Log states are exactly **`listed`** (indexed + reachable) and **`unlisted`** (reachable-by-slug, excluded from the index) — consistent with `03`. **Every fetched log is public**; draft privacy rests on slug unguessability. No authenticated-preview third state: inventing one would force persona to send a credential to render it — exactly the client-bundle-token trap this ticket exists to avoid. "Preview" = fetch the unlisted slug; "too private" = don't upload it yet.

### 4. Third-party secrets — invariant here, mechanism in `deed`

Steam/GitHub keys **and the custodian token hash** are **custodian-side only**, read from custodian's **process environment/config at startup**, never compiled into persona or sent to any browser. persona never calls Steam/GitHub even via a proxy: the poller (`02`/`09`) fetches server-side → SQLite cache → persona reads only `GET /v1/integrations/{source}`. **This ticket sets the invariant; `deed` picks the concrete secret store** (env var vs SSM SecureString vs systemd `EnvironmentFile`), inheriting `01`'s "nothing long-lived on disk" pressure. custodian treats the *source* of its environment as opaque, which is what lets deed swap mechanisms without touching custodian's code. (SSM SecureString with the default `aws/ssm` KMS key was costed at ~£0/mo — standard params + standard throughput free, KMS API usage fractions of a penny — so cost does not decide this; deed does.)

### 5. Rotation & revocation — single token, replace-and-restart

**One active token.** Rotate = generate a new token → update the secret via deed's delivery path → restart; **revocation is implicit** (old hash stops matching). No token registry (multi-user machinery) and **no self-credential endpoints** on custodian (which would be their own attack surface). No overlapping-token grace period — the restart blip is a non-issue for a hand-driven CLI. **Leak detection** is admin-surface access logging (timestamp, real IP, path, result); any `/admin/*` call you didn't make is the alarm — **requirement handed to `11`**, not a bespoke detector here.

### 6. Rate limiting & abuse — two threats, two layers

- **Origin hammering → nginx `limit_req`.** Lives at the reverse proxy on the `t4g.micro` (`01`), keeping custodian's Go free of throttling. Per-IP token bucket, **per-location** (tighter on `/admin/*`, looser on `/v1/*`), **real-client-IP-aware** via CloudFront `X-Forwarded-For` (`set_real_ip_from` + `real_ip_header`) — else it throttles all traffic as one edge IP. Returns `429`. CloudFront edge cache (`09`'s `ETag`/`Cache-Control`/`s-maxage`) absorbs the bulk; nginx is the backstop for cache-busting traffic that reaches origin.
- **Wallet-DoS on edge-cached static → AWS WAF rate-based rule (Option B).** nginx does **nothing** for the CloudFront bill — edge-cached hits never reach origin. An unmitigated flood (100M req / ~20 TB egress) was costed at **≈ $1,678 for one month**, almost all egress. **AWS Budgets cannot cap this** — it only alerts or revokes IAM perms, it cannot throttle CloudFront traffic. A **WAF rate-based rule on the distribution** is the only control that acts *during* an attack (drops abusive IPs at the edge, before cache/egress), converting ~$1,678 → ~$66 for the attack month at **~$6/mo steady-state**. **Chosen: Option B — WAF rate-based rule + an AWS Budgets alarm as the detect-and-notify floor.** Costing detail in [`docs/research/aws-cloudfront-wallet-dos-pricing.md`](../../../docs/research/aws-cloudfront-wallet-dos-pricing.md) (CloudFront per-unit figures corroborated via third-party trackers — spot-check at deploy).
- **Alarms** for an abnormal request-rate / `429` / spend uptick on **both** surfaces → **handed to `11`**.

### Ripples

- **Budget tension → `01`.** WAF's ~$6/mo pushes the total to **~£15/mo, over `01`'s ~£10 target** — a deliberate, blessed breach buying a hard ceiling on catastrophic spend. Recorded for `01` to revise.
- **Handed to `11` (observability):** admin-surface access logging as leak detection; alarms on abnormal request-rate / `429` volume / spend, on both public and admin surfaces.
- **Handed to `deed`:** the concrete secret store for the token hash + Steam/GitHub keys (constrained to never reach the client); Terraform-provisioning of the WAF rate-based rule and the Budgets alarm.
- **New fog (planning-only map, so not resolved here):** a **detailed monthly cost-expectation breakdown + Budgets-provisioning task at implementation time** — the full model across EC2 + S3 + CloudFront + WAF + KMS/SSM. Belongs to `deed`'s build phase; graduates when deed's provisioning boundary firms up.
- **No new decision tickets surfaced** — everything above lands on existing tickets (`01`/`11`/`deed`) or fog. `broom`'s `login`/`logout`/token-config detail is `17` (cli UX) implementation detail, already in that ticket's scope.
