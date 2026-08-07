# Spec: deed

Status: ready-for-agent

The Terraform component: the authoritative, declared record of the ground the
playground sits on. `deed` provisions the durable cloud shell every other
component runs on — the EC2 box, the S3 buckets, the CloudFront distribution, the
ECR repo, and the IAM identity that ties them together — and nothing else. It
does not build, deploy, or configure the application layer; a deploy does that.
`deed` is the one component capable of destroying every other one, so its apply
authority is a security decision, not a convenience.

Like `blank`, `deed` carries the property that it must be *capable* of
provisioning projects beyond the playground — its interface must not become
playground-shaped — while actually standing up other projects stays out of
scope.

This spec synthesises the resolved decision tickets `18` (state backend & apply
authority), `19` (provisioning boundary), and `20` (multi-project layout),
grounded in ADR-0001 (`01`, hosting & deployment posture), the monorepo tooling
(`13`), and the contracts `deed` provisions *for*: custodian's storage (`08`),
auth (`10`), and observability (`11`). Where a later ticket revised an earlier
one, the revised position is stated here as the single source of truth. `deed`
provisions the shell other components' specs depend on; where this spec names a
custodian/persona concern it records only the *provisioning* of it, and the
authoritative application contract lives in that component's spec.

## Problem Statement

ADR-0001 committed the playground to real AWS infrastructure — an EC2 `t4g.micro`
running custodian, S3 buckets for SQLite backups and media, a private persona
bucket, one CloudFront distribution fronting both, ACM, Route 53, and an IAM
instance profile. That is enough moving parts that clicking through the console
would be unrepeatable, undocumented, and impossible to reason about or recover.
The playground exists to *learn* infrastructure, so the ground it sits on wants
**declaring**, not clicking — which is what made `deed` a component in the first
place.

Two sharp constraints shape the problem:

- **`deed` can destroy everything.** It holds the one identity with authority over
  every durable resource, including the buckets that hold data ADR-0001 says to
  rent because "if it's gone, it's gone." A careless apply, a leaked credential,
  or an unmergeable committed state file could take unrecoverable data with it.
- **The credential principle from ADR-0001 must carry through.** Custodian
  deliberately holds no long-lived AWS credentials; `deed` adopting a static
  access key — for CI or for local apply — would be inconsistent with the
  decision that justified choosing AWS at all (the `NEXT_PUBLIC_NOTION_KEY`
  mistake in a different file).

And a scope tension: `deed` will one day own deployments for projects outside the
playground, which is the forcing function against its interface going
playground-shaped — but building multi-project machinery now would be speculative
ceremony against imagined consumers.

## Solution

`deed` is a **single in-repo Terraform configuration** (HCL) that provisions **the
durable shell** for the playground and nothing above it. The line is drawn as
**`deed` provisions the durable shell; a deploy owns the app layer** — a mutable,
long-lived box carrying an immutable app layer on top (containers are cattle; the
box and its SQLite volume are pets).

State lives in **S3 with native locking** (`use_lockfile`, Terraform 1.10+),
**split into multiple state files by logical component**, **never committed to
git**, in a **state bucket bootstrapped by hand once and referenced, never
managed by Terraform**. Apply is **laptop-only under short-lived IAM Identity
Center (SSO) credentials**; **CI never applies** — deed's CI stays capped at
`fmt` / `validate` / `plan` per `13`. Drift is **accepted as personal
responsibility** — no scheduled plan, because that would reintroduce the standing
AWS credential surface just eliminated.

`deed` provisions: the EC2 `t4g.micro` + EBS with a `user_data` bootstrap that
stops at "Docker engine running"; the named Docker volume the SQLite file lives
in; the ECR repo + lifecycle policy + instance-profile pull grant; SSM
`SecureString` parameters for the *bootstrap* secrets only (admin-token hash,
Grafana OTLP credential) + a path-scoped read grant; the private persona S3 bucket
(OAC-locked) and the one CloudFront distribution fronting both persona and the
API; a `deed`-owned CloudFront Function rewriting `/logs/<slug>/` →
`…/index.html`; and the IAM instance profile itself. It applies
`prevent_destroy` to the data buckets as a layout-independent safety invariant.

A **deploy** — laptop-driven under SSO, *not* `deed` — owns `compose.yml` +
`nginx.conf`, builds and pushes the custodian image to ECR, triggers a custodian
rollout via `just deploy-custodian` → SSM Run Command (no SSH), and publishes
persona via `s3 sync` + a CloudFront invalidation.

The multi-project shape is the **minimum that keeps `deed` from going
playground-shaped without building any machinery today**: flat HCL, no modules
(extract on the first *real* second consumer, rule of three), no published
distribution, no versioning — with **one deliberate investment**: **AWS
Organizations enabled now, the playground in its own member account**, taken
purely because moving stateful resources across accounts later is a genuine
migration whereas enabling the Org today is near-free. Project-two becomes a new
account, not a migration.

## User Stories

### Declaring and applying the shell

1. As the operator, I want the entire playground's durable infrastructure declared in HCL under `deed`, so that the ground the playground sits on is reproducible, reviewable, and recoverable rather than a pile of unrepeatable console clicks.
2. As the operator, I want to run `terraform plan` and see exactly what will change before any change is made, so that I never mutate infrastructure blind.
3. As the operator, I want to `terraform apply` from my laptop under short-lived SSO credentials, so that the one identity capable of destroying everything never corresponds to a static key sitting on disk.
4. As the operator, I want CI to run `fmt` / `validate` / `plan` but never `apply`, so that the repository proves the config is well-formed and shows the diff without ever holding apply authority or standing AWS credentials.
5. As the operator, I want drift I introduce by hand in the console to surface at my next local `apply`, so that I take personal responsibility for it without a scheduled plan reintroducing the credential surface I deliberately removed.

### State and its safety

6. As the operator, I want Terraform state stored in S3 with native locking, so that state is durable and concurrent applies are serialised without running a DynamoDB lock table.
7. As the operator, I want state split into multiple files by logical component (e.g. shared edge/DNS, custodian's box, the static site), so that the blast radius of any one apply or destroy is bounded to its component.
8. As the operator, I want the local `.tfstate` git-ignored and state never committed, so that state — which is plaintext secrets and unmergeable JSON — never lands in this repo's clean public history.
9. As the operator, I want the state bucket created by hand once (make-bucket + versioning + encryption + block-public-access) and merely referenced by the backend, never managed by Terraform, so that a botched refactor or destroy can never delete the bucket holding its own state.
10. As the operator, I want the hand-bootstrap steps documented in `deed`'s README, so that the one deliberately-imperative step is repeatable and legible.
11. As the operator, I want `prevent_destroy` on the data buckets (SQLite backups + media), so that no component destroy can silently take unrecoverable data regardless of how state is later re-partitioned.

### The compute shell

12. As the operator, I want `deed` to provision the EC2 `t4g.micro` + EBS root volume, so that custodian has a long-lived box to run on per ADR-0001.
13. As the operator, I want `user_data` to install Docker + the compose plugin + the AWS CLI and stop at "Docker engine running", so that the box is ready to accept an app layer without `deed` reaching up into the application.
14. As the operator, I want `deed` to provision the named Docker volume the SQLite file lives in, so that custodian's data survives container recreation while staying a pet the box owns.
15. As the operator, I want the box's AWS access to come solely from an IAM instance profile, so that there is never a long-lived AWS key on the box.

### The container registry

16. As the operator, I want `deed` to provision an ECR repository, so that custodian's image lives in the only registry the box can authenticate to with zero static credentials (instance-profile pull).
17. As the operator, I want an ECR lifecycle policy (keep last N, expire untagged), so that image storage stays flat and cheap as versions accumulate.
18. As the operator, I want the instance profile to carry an ECR pull grant, so that the box pulls images without a long-lived registry PAT (the reason GHCR was rejected).

### Edge, static site, and DNS

19. As the operator, I want `deed` to provision the private persona S3 bucket locked to CloudFront via OAC, so that the static site is served only through the edge and clients cannot bypass CloudFront to hit the bucket directly.
20. As the operator, I want `deed` to provision the single CloudFront distribution that fronts both persona and the custodian API, so that blogs cache at the edge and the API is same-origin per ADR-0001.
21. As the operator, I want `deed` to own a CloudFront Function that rewrites `/logs/<slug>/` → `…/index.html`, so that persona's directory-format clean URLs resolve against the private OAC bucket (which has no S3 website endpoint).
22. As the operator, I want CloudFront's public TLS certificate issued through ACM, so that AWS holds the certificate's private key and no cert material lives on disk.
23. As the operator, I want DNS delegated to Route 53 (registration staying at Squarespace per ADR-0001), so that the edge and origin records are declared alongside the rest of the shell.

### Secret provisioning (authorization, not values)

24. As the operator, I want `deed` to provision SSM `SecureString` parameters for the bootstrap secrets only — custodian's admin-token hash (`10`) and the Grafana OTLP credential (`11`) — so that the fixed startup-time identity secrets have a durable, encrypted home.
25. As the operator, I want `deed` to grant the instance profile a path-scoped read over those parameters, so that adding a bootstrap secret needs no policy edit.
26. As the operator, I want `deed` to deliver *authorization* to read secrets, never the runtime injection of them, so that custodian stays env-only and AWS-agnostic (the deploy wrapper does the SSM→tmpfs-env step, not `deed`).
27. As the operator, I want integration secrets (Steam, GitHub PAT, future) to be explicitly *not* provisioned by `deed`, so that the growing operational set lives in custodian's own SQLite, written through `broom`'s authed API and runtime-read with no stack restart (`19` revises `10`/`03`).
28. As the operator, I want bootstrap-secret values supplied via a git-ignored tfvars file (accepting they land in encrypted TF state per `18`), so that the value never enters git while the provisioning stays declarative.

### The deploy boundary (what deed does NOT do)

29. As the operator, I want `compose.yml` and `nginx.conf` to be application artifacts owned by a deploy, not `deed`, so that a route change or version bump never forces a `terraform apply` under `18`'s rare SSO-gated authority.
30. As the operator, I want to build custodian's image locally and `docker push` to ECR under SSO creds, so that shipping a new custodian version does not touch Terraform.
31. As the operator, I want `just deploy-custodian` to roll custodian out via SSM Run Command (`docker compose pull && up -d` on the box) with no SSH key and no inbound SSH, so that deploys ride the instance profile + SSO rather than a standing key.
32. As the operator, I want `just deploy-persona` to build, `s3 sync`, and issue a CloudFront invalidation under SSO, so that publishing the static site is a laptop gesture with no CI or OIDC role in v1.
33. As the operator, I want systemd to largely dissolve into Docker restart policies (at most a one-line boot shim), and Litestream to run as a sidecar container on the SQLite volume, so that the box's operational surface is Docker, not hand-maintained units (`19` revises `01`).

### Multi-project capability (capable-of, don't-build)

34. As the operator, I want `deed` to be a single in-repo config with flat HCL and no modules today, so that "capable of other projects" is preserved (flat HCL forecloses nothing; extraction is mechanical) without building speculative machinery.
35. As the operator, I want module extraction deferred to the first *real* second consumer (rule of three), so that any module interface is designed against two concrete examples rather than one imagined one.
36. As the operator, I want AWS Organizations enabled now with the playground in its own member account, so that a second project becomes a new account rather than a painful cross-account migration of stateful resources.
37. As the operator, I want each member account to hand-bootstrap its own state bucket, so that a blast in one project's state cannot touch another's and per-account SSO creds cannot reach another account's state.
38. As the operator, I want no published-module distribution and no versioning in v1, so that nothing is pinned that nothing consumes; both revive only when a real second repo needs `deed`'s modules.

## Implementation Decisions

### Tool and backend (`18`)

- **Terraform**, not OpenTofu — chosen for career-alignment (the industry-default
  HCL toolchain, mirroring `07`'s Go logic). BUSL doesn't bite at this scale, and
  OpenTofu's drop-in compatibility means the skill transfers if ever wanted.
- **Backend: S3 with native locking** (`use_lockfile = true`, Terraform 1.10+).
  State lives in the same rented-durability layer as `08`/`11`; the conditional-
  write lock object replaces the legacy DynamoDB lock table (one fewer moving
  part). HCP / hosted backend rejected — it reintroduces a SaaS dependency and a
  long-lived-credential surface against ADR-0001.
- **State never committed to git.** Local `.tfstate` is `.gitignore`'d; S3
  versioning + encryption-at-rest hold the real copy. State is plaintext secrets
  and unmergeable JSON — exactly the "if it's gone, it's gone" data ADR-0001 says
  to rent, not self-manage in a public repo.

### Bootstrap and apply authority (`18`)

- **State bucket bootstrapped by hand once, then referenced — never managed.**
  ~4 CLI commands (make-bucket + versioning + encryption + block-public-access),
  documented in `deed`'s README, kept *outside* Terraform's reach and explicitly
  **not** `terraform import`ed. Managing the bucket that holds its own state is a
  footgun; this is the one place clicking beats declaring.
- **Apply is laptop-only under short-lived IAM Identity Center (SSO) creds.** No
  static IAM access key on disk for the identity that can destroy everything —
  the sharpest case of ADR-0001's "nothing long-lived on disk."
- **CI never applies.** deed's CI is capped at `fmt` / `validate` / `plan`
  (`13`). **GitHub OIDC for a future gated CI-apply is parked as fog** — revived
  only if apply ever leaves the laptop.

### Blast radius and the data-bucket invariant (`18`)

- **Multiple state files, split by logical component** (e.g. shared edge/DNS,
  custodian's box, the static site) — not one monolith. **Exact boundaries and
  where the shared CloudFront/Route 53/ACM land are left to implementation time**;
  they are not worth pre-deciding.
- **The one layout-independent invariant:** `lifecycle { prevent_destroy = true }`
  on the data buckets (SQLite backups + media), so no component destroy can
  silently take unrecoverable data regardless of how state is partitioned.

### The provisioning boundary — what `deed` provisions (`19`)

The durable shell, and only the durable shell. Everything runs in **Docker via
`docker compose` on one EC2 `t4g.micro`** — **ECS rejected** (Fargate breaks
SQLite-on-local-disk, EFS/NFS is a file-locking corruption hazard, and it costs
~3×; ECS-on-EC2 is ceremony for a single box).

- **EC2 `t4g.micro` + EBS**, with **`user_data`** installing Docker + compose
  plugin + AWS CLI — bootstrap stops at "Docker engine running." `user_data`, not
  a pre-baked AMI: a build pipeline earning its keep on a boot you do ~never; an
  AMI stays a clean later optimisation.
- **The named Docker volume** the SQLite file lives in (survives container
  recreation).
- **ECR repo + lifecycle-retention policy** (keep last N / expire untagged) +
  **instance-profile pull grant**. Layer dedup keeps a static-Go rebuild at
  ~15 MB/version; the policy caps storage flat. ECR beat GHCR on cost *and*
  zero-static-cred pull (instance profile) — GHCR's private pull needs a
  long-lived PAT on the box (the `NEXT_PUBLIC_NOTION_KEY`-class mistake). Pricing
  in [`docs/research/ecr-vs-ghcr-pricing.md`](../../../docs/research/ecr-vs-ghcr-pricing.md).
- **SSM `SecureString` parameters for the bootstrap secrets only** + a
  **path-wildcard instance-profile read grant** (adding one needs no policy
  edit). Values from a **git-ignored tfvars** → they land in encrypted TF state
  (accepted per `18`).
- **The private persona S3 bucket** (OAC-locked to CloudFront) and **the one
  CloudFront distribution** fronting both persona and the API (ADR-0001).
- **A `deed`-owned CloudFront Function** rewriting `/logs/<slug>/` →
  `…/index.html` — a mainstream AWS pattern required because `15`/`19` use a
  private OAC bucket (not a public S3 website endpoint, which would let clients
  bypass CloudFront and undercut `10`/`19`'s perimeter). This is a `deed` line
  item surfaced by `15`.
- **The IAM instance profile** itself, and CloudFront's public TLS cert via
  **ACM** (AWS holds the key). **DNS via Route 53** (registration stays at
  Squarespace).

### The provisioning boundary — what a deploy does, not `deed` (`19`)

A **mutable long-lived box carrying an immutable app layer** (cattle on a pet).
All laptop-driven under SSO; **no CI, no OIDC role in v1**.

- **Owns `compose.yml` + `nginx.conf`** — they version with custodian and carry
  the image tag, so they are application artifacts. Putting them in Terraform
  would force a `terraform apply` per route change / version bump, colliding with
  `18`'s rare SSO-gated apply. nginx is the origin reverse-proxy + `limit_req`
  (`10`), not TLS termination.
- **Builds the image locally → `docker push` to ECR** under SSO creds. Builds are
  local for now.
- **`just deploy-custodian` → SSM Run Command** → the box runs a **deploy
  wrapper**: fetch bootstrap secrets **SSM→env onto tmpfs (`/run`)**, then
  `docker compose pull && up -d`. **No SSH key, no inbound SSH.**
- **`just deploy-persona`** → build (baking profile from custodian's public read
  API) → `aws s3 sync` → CloudFront `create-invalidation --paths "/*"` (free
  under 1,000/mo), all under SSO.
- **`systemd` largely dissolves** into Docker `restart: unless-stopped` (at most a
  one-line boot shim runs the wrapper). **Litestream** (`08`/`11`) becomes a
  **sidecar container** on the SQLite volume, keeping custodian's image a pure
  static binary.

### The secret model (`19`, revises `10`/`11`/`03`)

Two classes, split by whether the secret can live behind custodian's own authed
API:

- **Integration secrets (operational, growing set)** — Steam, GitHub PAT, future.
  **Live in custodian's own SQLite** (`08`), **`broom`-written via the authed
  admin API**, **runtime-read on each poll → no stack restart**, **not
  AWS-specific**. `deed` provisions *nothing* for these. A self-hosted secrets
  manager (Vault/OpenBao/Infisical) was **rejected** — a 24/7 component
  reintroducing a bootstrap token that SSM + instance-profile currently avoids.
- **Bootstrap/identity secrets (fixed set, startup-time)** — custodian's
  **admin-token hash** (`10`, can't live in the DB it authorizes) and the
  **Grafana Cloud OTLP credential** (`11`). Delivered **env-at-startup via the
  SSM→env path**. **`deed` provisions the param + read grant, never the runtime
  injection.**

**Injection mechanism (why custodian stays env-only *and* AWS-agnostic):** `deed`
delivers *authorization*, not the value. On a plain compose-on-EC2 box Terraform
has no runtime hook, so the **deploy wrapper** (not `deed`) does
`aws ssm get-parameters-by-path --path /custodian/ --with-decryption`, writes
`KEY=value` to a tmpfs env file (`/run`, RAM-only), and compose forwards it via
`env_file:`; custodian reads only `os.Getenv`. **Honest caveat:** Docker persists
a container's env into its root-only on-disk config, so "zero secret bytes on
disk" isn't literally true — but it is strictly smaller exposure than the
plaintext-in-state already accepted, and truly-zero-on-disk would require
custodian to self-fetch (rejected, breaks env-only).

**Non-secrets by design (so they aren't reintroduced):** box→AWS = instance
profile; deed apply = SSO short-lived (`18`); any future CI→AWS = GitHub OIDC
short-lived. **No long-lived AWS keys anywhere.** `broom`'s plaintext bearer
token stays client-side in its XDG `0600` config (`10`/`17`).

### Multi-project layout (`20`)

- **Consumption model — single in-repo config.** Published-module distribution
  (the `blank` "consumed from another repo" pattern) is **fog**, graduating only
  when a real second repo needs it. The forcing function against
  playground-shaping is a clean module interface, not the act of publishing — so
  publishing waits.
- **Modules — none yet; flat now, extract on real reuse.** Straightforward
  playground HCL (`ec2`/`cloudfront`/`s3`/…), no `modules/`↔`projects/`
  indirection. Flat HCL forecloses nothing (extraction is mechanical), so
  "capable of other projects" holds; deferred to the first *real* second
  consumer (rule of three).
- **Versioning — moot.** Nothing published ⇒ nothing to pin.
- **AWS accounts — Organizations now, playground in its own member account.** The
  one deliberate "more moving parts" call, on retrofit-pain: cross-account moves
  of S3/CloudFront/ACM/Route 53 are real migrations; enabling the Org today is
  near-free. **`18`/`19`'s single-account SSO framing is now the playground's own
  member account.**
- **Cross-project state — state bucket per account.** Each member account
  hand-bootstraps its own state bucket (repeating `18`'s gesture per account).
  **`18`'s per-component state split is unchanged *inside* the playground
  account.**

### Monorepo placement (`13`)

- **`deed` lives in-repo** at the flat, named top level
  (`custodian/ broom/ blank/ persona/ deed/`) but **nothing applies it from CI
  today** — CI caps at `fmt` / `validate` / `plan`.
- **Terraform state is never committed** (reinforces `18`).
- The thin root `justfile` delegates to Terraform for deed's recipes; a gated
  CI-apply + the GitHub OIDC role it would need are future fog.

## Testing Decisions

**The single seam: plan-as-contract.** `deed` has no runtime to exercise and its
CI ceiling is already `fmt` / `validate` / `plan` (`13`/`18`), so the observable
behaviour under test is **"the configuration is well-formed and the plan against
the real S3 backend is clean and matches expectations, with the safety invariants
present."** Tests assert on the *plan and the config's declared guarantees* —
never on Terraform's internal graph or provider internals.

**What makes a good `deed` test:**

- It asserts external, operator-visible properties: `terraform fmt -check` passes,
  `terraform validate` passes, `terraform plan` produces **no unexpected changes**
  (a clean plan on an applied config, or an expected diff on a deliberate change),
  and the plan/config carries the **layout-independent invariants** — most
  importantly `prevent_destroy = true` on the data buckets, the persona bucket
  being private + OAC-locked (no public-website endpoint), the box's access coming
  only from an instance profile, and no long-lived AWS key resource existing
  anywhere.
- It never asserts on which module a resource lives in, resource addresses, or
  other implementation details that `20`'s "extract on real reuse" will churn.

**What is tested:** the playground root configuration(s), through `plan` against
the real (hand-bootstrapped) S3 backend under SSO creds. Because state is split
per logical component (`18`), each component's config is plan-checked
independently, matching how it is applied.

**Explicitly not adopted in v1:** native `terraform test` (`.tftest.hcl`
attribute assertions) and third-party policy engines (checkov/tflint beyond
`fmt`/`validate`). `20` defers modules until a real second consumer, so
module-unit assertions have nothing stable to bind to yet; they graduate with the
first extracted module. Static-analysis-only (no plan-against-backend) was
rejected because it would not exercise the actual state backend or confirm the
`prevent_destroy` invariant holds against real state.

**Prior art:** there is no existing `deed` test suite — this spec establishes the
pattern. It should follow idiomatic Terraform CI: `fmt -check`, `validate`, and a
`plan` step wired into the root `justfile` (`13`) and mirrored in CI, with the
plan run under SSO creds against the referenced (never-managed) state bucket.

## Out of Scope

- **Building, deploying, or configuring the application layer.** `compose.yml`,
  `nginx.conf`, image builds, `just deploy-custodian` / `deploy-persona`, and the
  SSM→tmpfs-env injection are a **deploy**'s job, not `deed`'s (`19`). This spec
  records only the *provisioning* that makes them possible.
- **CI apply and the GitHub OIDC role.** Fog until apply leaves the laptop; deed
  CI stays `fmt`/`validate`/`plan` (`13`/`18`).
- **Scheduled drift detection / read-only plan on a timer.** Rejected — it would
  reintroduce the standing AWS credential surface `18` eliminated, to solve a
  multi-operator problem a solo playground doesn't have. Rides the same future
  OIDC path if it ever graduates.
- **Published-module distribution + its versioning** (`20`). Flat in-repo HCL in
  v1; module extraction and semver-pinned consumption from another repo graduate
  only when a real second consuming repo exists (rule of three) — mirroring
  `blank`'s "publishable, but adopting it elsewhere is out of scope."
- **Actually provisioning non-playground projects.** `deed` must be *capable* of
  it (interface not playground-shaped), but standing up other projects'
  infrastructure is their own work (ADR-0001 out-of-scope ruling, kin to `blank`).
- **A pre-baked AMI + its build pipeline.** `user_data` is chosen for a
  ~never-run boot; an AMI is a clean later optimisation, not v1 (`19`).
- **Integration-secret provisioning** (Steam/GitHub PAT). Those live in
  custodian's SQLite, `broom`-written and runtime-read (`19` revises `10`/`03`);
  `deed` provisions nothing for them.
- **CloudFront↔origin perimeter security** — the shared-secret header the origin
  verifies so it can't be reached bypassing the edge/WAF, plus origin TLS for
  CloudFront→nginx (Let's Encrypt/certbot). A separate edge/origin-security
  discussion (kin to `10`'s WAF work), deliberately parked as fog by `19`; it will
  span `deed` (the CloudFront custom header) and the deploy (nginx config) when it
  graduates.
- **The detailed monthly cost model + Budgets-provisioning task** (from `10`) —
  the itemised EC2 + S3 + CloudFront + WAF + KMS/SSM breakdown and the AWS Budgets
  alarm + WAF rate-based rule provisioning. It belongs to `deed`'s build phase but
  is its own fog ticket, not this spec.
- **The Netlify→AWS cutover sequencing** (TTL lowering, cert validation before the
  switch, avoiding a no-site window). ADR-0001 settled where DNS lands; the
  sequencing is a separate effort.

## Further Notes

- **`deed` provisions authorization, never runtime values.** The single clearest
  through-line of the secret model: `deed` creates the SSM parameter and the
  read grant; the *value* arrives via a git-ignored tfvars into encrypted state,
  and the *injection* into custodian's env is the deploy wrapper's job on the box.
  Custodian only ever calls `os.Getenv`. Keep that boundary — do not teach `deed`
  to inject, and do not teach custodian to self-fetch.
- **The `/media/` and `/logs/` path shapes both land at the edge.** The single
  CloudFront distribution's root is namespaced by path prefix: `/media/<key>` for
  media (custodian origin, `08`/`17`), `/logs/<slug>/` for blog pages (persona
  bucket, rewritten to `…/index.html` by the `deed`-owned CloudFront Function per
  `15`). `deed` owns the distribution + the function; the origins' content is not
  `deed`'s.
- **The "capable-of, don't-build" discipline is load-bearing three times over.**
  It appears in `blank` (publishable, not adopted), in the custodian BaaS fog, and
  here — flat HCL + Organizations-now is the *minimum* investment that keeps the
  interface honest without speculative modules. When a trade-off arises, prefer
  the option that forecloses nothing over the one that builds machinery for an
  imagined second consumer.
- **`prevent_destroy` on the data buckets is the invariant that survives every
  refactor.** `18` deliberately left state boundaries to implementation time, but
  this one guard must be present no matter how the state files are partitioned —
  it is the last line against `deed`'s destroy authority taking unrecoverable
  data.
- **This spec revises earlier tickets where `19`/`20` did:** `01`'s
  "nginx + systemd by hand" → Docker + compose (`19`); `18`/`19`'s single-account
  framing → the playground's own member account under an Org (`20`); `10`/`03`'s
  env-var integration secret → custodian's DB, runtime-read (`19`). Those revised
  positions are stated here as the single source of truth for `deed`.
