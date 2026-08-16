# 07 — Operator deploy: build & push the image, then roll custodian out

**What to build:** The operator's laptop-driven path for shipping a custodian
version — the **deploy** actor the spec keeps distinct from both `deed` (Terraform)
and custodian (the Go service). All under SSO, **no CI, no OIDC role in v1**
(`deed.md:239`). Shipping a version never touches Terraform (`deed.md:148`,
story 30). This is the natural home for the `just` push recipe: the deploy owns
`compose.yml` + `nginx.conf` (application artifacts that version with custodian
and carry the image tag — `deed.md:241-243`), builds the image locally and
`docker push`es it to the ECR repo `deed/03` provisioned, then triggers a rollout
via `just deploy-custodian` → **SSM Run Command** (no SSH) → a **deploy wrapper**
on the box that reads bootstrap secrets **SSM→env onto tmpfs (`/run`)** and runs
`docker compose pull && up -d` (`deed.md:248-250`, `276-281`).

Two seams, split by dependency so the push half lands as soon as ECR exists:

- **Build & push** (unblocked by `deed/03` + `custodian/09`): resolve the registry
  URL from `deed/compute`'s `ecr_repository_url` output, `docker login` to ECR
  under SSO, build/tag `linux/arm64`, `docker push`. A `just` recipe.
- **Rollout** (wants the edge, `deed/05`, since `nginx.conf` is an origin
  reverse-proxy + `limit_req` behind CloudFront — `deed.md:244-245`):
  `compose.yml` + `nginx.conf`, the SSM→tmpfs deploy wrapper, and
  `just deploy-custodian` via SSM Run Command.

`deed` provisions *authorization*, never the runtime injection of secret values
(`deed.md:273-274`); the deploy wrapper does the SSM→env step, not `deed`.

**Blocked by:** 03, 05, custodian/09.

**Status:** done

- [x] `just` recipe builds custodian's `linux/arm64` image, logs into ECR under SSO, tags, and `docker push`es it — no Terraform touched
- [x] Registry URL comes from `deed/compute`'s `ecr_repository_url` output (single source of truth), not a hand-duplicated string
- [x] `compose.yml` + `nginx.conf` live as deploy artifacts carrying the image tag; nginx is origin reverse-proxy + per-location `limit_req`, not TLS termination
- [x] Deploy wrapper fetches bootstrap secrets SSM→env onto tmpfs (`/run`), then `docker compose pull && up -d`; no secret persisted to disk beyond Docker's own root-only config (accepted caveat)
- [x] `just deploy-custodian` rolls out via SSM Run Command — no SSH key, no inbound SSH
- [x] Litestream runs as a sidecar container on the SQLite volume, keeping custodian's image a pure static binary

Implemented as `deploy/` (compose.yml, nginx.conf, litestream.yml, deploy-wrapper.sh)
plus `just push-custodian` and `just deploy-custodian`. `just --list` parses; the
rendered rollout script and the artifact tar round-trip locally.

**Human-verification deferred:** an end-to-end push + rollout needs SSO creds, the
applied `deed/compute` outputs, and the live box — not runnable from CI or this
laptop without an AWS session. The push half is exercisable as soon as an SSO
session and a built image exist; the rollout half additionally needs the box up
with the bootstrap secrets provisioned.
