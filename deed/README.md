# deed

The declared record of the ground the playground sits on. Flat in-repo HCL (no
modules), one component per directory, each its own Terraform root wired to a
shared S3 backend.

## Layout

```
deed/
  foundation/   account wiring + backend scaffold (creates nothing)
  compute/      custodian's box: EC2 + EBS, instance profile, SSM bootstrap secrets, ECR registry, data buckets, the edge (CloudFront + ACM + Route 53)
```

## `compute`

custodian's box and the identity it boots with. An EC2 `t4g.micro` on a gp3
root volume whose `user_data` installs Docker, the compose plugin, and the AWS
CLI and stops at "Docker engine running" — plus the named Docker volume the
SQLite file lives in. The box's only AWS identity is an IAM instance profile;
there is no long-lived AWS key on it.

The bootstrap secrets custodian reads at startup are SSM `SecureString`
parameters under a shared prefix, their values supplied via git-ignored tfvars —
copy `compute/terraform.tfvars.example` to `compute/terraform.tfvars` and fill in
real values. They land in encrypted state, accepted. The instance profile carries
a **path-wildcard read** over that prefix, so adding a bootstrap secret later is a
tfvars edit, not a policy edit. `deed` delivers authorization to read; the deploy
wrapper does the SSM→env step on the box.

### The registry

custodian's image lives in an ECR repository the box pulls from **via its instance
profile alone** — no long-lived registry PAT on the box (the reason GHCR was
rejected). The instance profile carries an ECR pull grant: repository-scoped
layer/image reads plus the registry-level `ecr:GetAuthorizationToken` (which cannot
be resource-scoped). A lifecycle policy keeps image storage flat as versions
accumulate — it retains the most recent `ecr_image_retention_count` images and
expires untagged ones after a day.

### The data buckets

Two S3 buckets are the durable homes ADR-0001 says to rent: the **media bucket**
custodian serves uploads from and the **SQLite-backup bucket** Litestream
replicates the database to. Both are fully private (neither is reached directly —
media flows through the custodian origin, the backup is internal) and both carry
`lifecycle { prevent_destroy = true }`, so no component destroy can silently take
unrecoverable data regardless of how `deed`'s state is later split. The safety
invariant lives on the resource, not on the state layout. The instance profile
carries a read/write grant (`GetObject`/`PutObject`/`DeleteObject` plus
`ListBucket`) over both, covering custodian's media reserve/confirm flow and
Litestream's backup.

### The edge

The single CloudFront distribution the whole playground fronts, with the
custodian box as its API origin (ADR-0001). Viewers reach custodian over HTTPS at
`var.custodian_domain_name` (`custodian.mihirsingh.dev`); the apex is persona's
front-facing website, added in ticket 06. The distribution's public certificate is
issued through **ACM in us-east-1** — the only region CloudFront reads viewer
certificates from — so AWS holds the private key and no certificate material lands
on disk. The default cache behavior forwards the whole viewer request except the
Host header and does not cache, so custodian's auth and cookies work through the
edge; persona's cacheable `/logs/` behavior is added in ticket 06.

The box origin needs a durable address, so it carries an **Elastic IP** with an
`origin.<custodian_domain_name>` A record; the edge<->origin hop is plain HTTP,
locked to CloudFront alone by pinning the box's security-group ingress to the
`com.amazonaws.global.cloudfront.origin-facing` managed prefix list.

DNS is a **Route 53 hosted zone** for `var.zone_name` (the apex — custodian and
persona both live under it); registration stays at Squarespace. **The zone's name
servers (`route53_name_servers` output) must be set at Squarespace before an apply
can finish** — ACM's DNS validation records resolve only once delegation is live,
and `aws_acm_certificate_validation` blocks until they do.

### The env-var contract

Each `bootstrap_secrets` key **is** the environment variable name custodian reads
(`custodian/internal/config/config.go`). The deploy wrapper reads every parameter
under the prefix and exports each one under its leaf name verbatim — no mapping
table. The secrets `deed` provisions:

| SSM parameter leaf (= env var) | Value |
|---|---|
| `CUSTODIAN_ADMIN_TOKEN_HASH` | hex-encoded SHA-256 of the admin bearer token |
| `CUSTODIAN_OTLP_ENDPOINT` | base `.../otlp` gateway URL (non-secret, co-located here so the one SSM→env path delivers it; custodian exports nothing when it is unset) |
| `CUSTODIAN_OTLP_AUTHORIZATION` | full `Authorization` header value, e.g. `Basic <base64(instanceID:token)>` |
| `CUSTODIAN_STEAM_KEY` | Steam Web API key |
| `CUSTODIAN_GITHUB_PAT` | GitHub PAT |

The remaining **non-secret** runtime config custodian needs is *not* stored in
`deed` — the deploy wrapper/compose set it directly: `CUSTODIAN_CORS_ALLOWLIST`,
`CUSTODIAN_MEDIA_BUCKET`, `CUSTODIAN_MEDIA_CDN_BASE`, and the optional
`CUSTODIAN_ADDR` / `CUSTODIAN_DB_PATH`.

`deed` provisions into the operator's existing single AWS account via ambient
SSO credentials — it does not enable an Organization or create a member account.
`foundation` creates no resources; it wires the backend and reports which account
the credentials resolved to (a wrong-account guard), so its `plan` is the clean,
no-change proof that the S3 backend works. Enabling an Org and splitting accounts
is deferred until a real second project needs it.

Every component is a sibling directory with its own backend `key`
(`<component>/terraform.tfstate`). This is the per-component state split: a
`destroy` in one component can never touch another's state. New components
follow the same shape — a flat root, its own `key`, no shared module.

## The one imperative gesture: bootstrap the state bucket by hand

The backend references an S3 bucket that Terraform **never creates, imports, or
manages** — a bucket managing the state that describes it is a bootstrap cycle.
Create it by hand once, then only reference it. Run the script against the
account that owns the state (short-lived SSO credentials, see below). It is
idempotent — safe to re-run.

Linux / macOS / Git Bash:

```sh
./scripts/bootstrap-state-bucket.sh
```

Windows PowerShell:

```powershell
./scripts/bootstrap-state-bucket.ps1
```

Both default to bucket `deed-tfstate-playground-euw2` in `eu-west-2` and enable
versioning, `aws:kms` encryption, and a full public-access block. Override with
`BUCKET=… REGION=… ./scripts/bootstrap-state-bucket.sh` or
`-Bucket … -Region …` on PowerShell.

The bucket name must match the `bucket` in each component's `backend "s3"`
block. State locking is native (`use_lockfile = true`) — no DynamoDB table.

## Credentials: SSO only

Apply is laptop-only under short-lived IAM Identity Center (SSO) credentials.
There is no static IAM access key on disk anywhere. Sign in before running any
recipe that touches AWS:

```sh
aws sso login --profile <your-sso-profile>
export AWS_PROFILE=<your-sso-profile>
```

CI runs `fmt` and `validate` only (see `.github/workflows/deed.yml`) — both need
no AWS credentials. `plan` and `apply` are kept local, run by hand under SSO
credentials; CI never touches the backend.

> **GitHub Actions is paused repo-wide right now.** Actions is disabled at the
> repository level (`repos/mihirs16/playground/actions/permissions` →
> `enabled: false`), so no workflow — including `deed.yml` — runs on any branch
> or PR. This is a deliberate hold: Terraform is planned and applied **locally
> only** until CI is revisited. The `deed.yml` file is left in place but inert.
> Re-enable with `gh api -X PUT repos/mihirs16/playground/actions/permissions -F enabled=true`.

## Recipes

From the repo root:

```sh
just deed-fmt         # gofmt-equivalent: canonical HCL formatting
just deed-validate    # config is internally consistent (no backend/creds needed)
just deed-plan        # plan against the real backend (needs SSO credentials)
just deed-apply       # apply a component (needs SSO credentials); CI never runs this
```

`deed-plan` and `deed-apply` take a component name, defaulting to `foundation`:

```sh
just deed-plan foundation
just deed-apply foundation
```
