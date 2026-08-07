# deed: provisioning boundary

Type: grilling
Status: resolved
Blocked by: 07, 13

## Question

Where is the line between what `deed` provisions and what a deploy does?

- **The instance** — `deed` creates the EC2 instance, but does it also configure it? A raw `user_data` script, a config-management tool, or a pre-baked AMI built in CI?
- **The nginx config and the systemd unit** — infrastructure or application? They change with custodian, not with the account, which argues they aren't `deed`'s.
- **The artifact** — does a deploy ship a binary onto a long-lived instance, or replace the instance entirely?
- **Secrets that aren't IAM** — the Steam and GitHub API credentials still exist and still need somewhere to live. Does `deed` provision SSM Parameter Store or Secrets Manager entries, and who writes the values into them?
- **The static site** — `deed` owns the bucket and the distribution, but who uploads persona's build output and issues CloudFront invalidations?

## Context

Surfaced while resolving `01`.

Blocked on `07` because the deployment artifact depends on the language — a Go static binary, a Node app with `node_modules`, and a container image are three different deploy stories. Blocked on `13` because this is half of the CI/CD shape, and `13` has to place that in the repo.

The sharp constraint is `01`'s SQLite-on-EBS decision. State living on the instance's own disk pushes against immutable infrastructure, because replacing the instance means moving or re-restoring the volume. Continuous replication to S3 makes that survivable — but the deploy model has to say so explicitly rather than discovering it during the first rebuild.

## Answer

The line is drawn as **`deed` provisions the durable shell; a deploy owns the app layer**, with a **mutable long-lived box carrying an immutable app layer on top** (containers are cattle, the box + SQLite volume are pets). Everything runs in **Docker via `docker compose`** on one EC2 `t4g.micro` — **ECS rejected** (Fargate breaks SQLite: no persistent local disk, EFS/NFS is a SQLite file-locking corruption hazard, and it costs ~3×; ECS-on-EC2 is pure ceremony for a single box).

### `deed` provisions (the durable shell)

- The **EC2 `t4g.micro` + EBS**, and **`user_data`** that installs **Docker + compose plugin + AWS CLI** — bootstrap stops at "Docker engine running." (`user_data`, not a pre-baked AMI: a build pipeline earning its keep on a boot you do ~never; AMI stays a clean later optimisation.)
- The **named Docker volume** the SQLite file lives in (survives container recreation).
- The **ECR repo + a lifecycle-retention policy** (keep last N / expire untagged) + the **instance-profile pull grant**. Layer dedup keeps a static-Go rebuild at ~15 MB/version; the lifecycle policy caps storage flat. Pricing confirmed in [`docs/research/ecr-vs-ghcr-pricing.md`](../../../docs/research/ecr-vs-ghcr-pricing.md) — ECR ~$0.10/mo, same-region pulls free, and the only registry the box can auth to with **zero static credentials** (instance profile). GHCR was rejected: private pull needs a long-lived PAT on the box (the `NEXT_PUBLIC_NOTION_KEY`-class mistake), and its $0 is an un-metered-today artifact that could become ~$13–15/mo.
- **SSM `SecureString` parameters for the bootstrap secrets only** (see secret model), values from a **git-ignored tfvars** → they land in encrypted TF state (accepted, per `18`) + the **instance-profile read grant** (path-wildcard so adding one needs no policy edit).
- The **persona S3 bucket** (private, OAC-locked to CloudFront) and the **CloudFront distribution** (the one distribution fronting both persona and the API, per `01`).
- The **IAM instance profile** itself.

### A deploy does (app layer, laptop-driven under SSO)

- **Owns `compose.yml` + `nginx.conf`** — they version with custodian and carry the image tag, so they are application artifacts, **not** `deed`'s (putting them in Terraform would force a `terraform apply` per route change / version bump, colliding with `18`'s rare SSO-gated apply). nginx is the origin reverse-proxy + `limit_req` (`10`), not TLS termination.
- **Build the image locally → `docker push` to ECR** under SSO creds (`aws ecr get-login-password`). **Builds are local for now** — CI does not build or publish.
- **`just deploy-custodian` → SSM Run Command** → the box runs the **deploy wrapper**: fetch bootstrap secrets **SSM→env onto tmpfs (`/run`)**, then `docker compose pull && up -d`. **No SSH key, no inbound SSH** (Run Command rides the instance profile + SSO). CI's job would end at "image in ECR" if/when it takes over.
- **`just deploy-persona`** → build (baking profile from custodian's public read API) → `aws s3 sync` → CloudFront `create-invalidation --paths "/*"` (free under 1,000/mo), all under SSO creds. **No CI, no OIDC role** — solo deployer, laptop-driven.
- **`systemd` largely dissolves** into Docker `restart: unless-stopped`; at most a one-line boot shim runs the wrapper. **Litestream** (`08`/`11`) becomes a **sidecar container** on the SQLite volume, keeping custodian's image a pure static binary.

### The secret model (the ticket's sharpest sub-decision)

Two classes, split by whether the secret can live behind custodian's own authed API:

- **Integration secrets (operational, growing set)** — third-party API keys (Steam, GitHub PAT, future). **Live in custodian's own SQLite** (`08`'s store), **`broom`-written via the authed admin API**, **runtime-read on each poll → no stack restart**, and **not AWS-specific** (custodian never learns a cloud secret store exists). This is "our own secrets manager" realised as *reusing the store already running* — a **self-hosted secrets manager (Vault/OpenBao/Infisical) was rejected**: a 24/7 RAM-hungry component that reintroduces a bootstrap token custodian must authenticate with (SSM+instance-profile currently needs *zero*), against every prior "one fewer moving part" call. Optional later: column-encrypt these rows with a single bootstrap master key.
- **Bootstrap/identity secrets (fixed set, needed at startup)** — custodian's **admin-token hash** (`10`, can't live in the DB it authorizes) and the **Grafana Cloud OTLP credential** (`11`, needed at startup to emit telemetry). Delivered **env-at-startup via the SSM→env path**. `deed` provisions the param + read grant, never the runtime injection.

**Injection mechanism (why custodian stays env-only *and* AWS-agnostic):** `deed` delivers *authorization*, not the value. On a plain compose-on-EC2 box Terraform has no runtime hook, so the **deploy wrapper** does `aws ssm get-parameters-by-path --path /custodian/ --with-decryption` (path-scan → no per-secret code), writes `KEY=value` to a **tmpfs env file** (`/run`, RAM-only), and compose forwards it via `env_file:`. custodian reads only `os.Getenv`. Honest caveat accepted: Docker persists a container's env into its root-only on-disk config, so "zero secret bytes on disk" isn't literally true — but it's a strictly smaller exposure than the plaintext-in-state already accepted, and truly-zero-on-disk would require custodian to self-fetch (rejected, breaks env-only).

**Non-secrets by design (stated so they aren't reintroduced):** box→AWS = instance profile; deed apply = SSO short-lived (`18`); any future CI→AWS = GitHub OIDC short-lived. **No long-lived AWS keys anywhere.** CloudFront's public TLS cert = **ACM** (AWS holds the key). `broom`'s plaintext bearer token stays client-side in its XDG `0600` config (`10`/`17`).

### Parked as fog (surfaced here)

- **CloudFront↔origin perimeter security** — shared-secret header so the origin can't be reached bypassing the edge/WAF, plus origin TLS for CloudFront→nginx. A separate edge/origin-security discussion (kin to `10`'s WAF work), not the deed-vs-deploy boundary.
- **CI build-and-publish + a GitHub OIDC role** — revived only if builds/deploys leave the laptop.

### Revises prior tickets

- **`10`**: integration keys move from "env var at startup" → **custodian's DB, runtime-read**; env-at-startup now carries only the fixed bootstrap set (admin-token hash + Grafana token).
- **`11`**: names the **Grafana OTLP credential** as a bootstrap secret needing startup delivery.
- **`03`**: the integration model's per-source secret becomes a **stored (DB) secret** rather than an env-var name.
- **`01`**: "long-lived EC2 running nginx + systemd by hand" → **Docker + compose**; systemd reduced to a boot shim.
