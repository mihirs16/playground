# 01 — Terraform foundation: single-account wiring, S3 state backend, CI plan gate

**What to build:** The ground `deed` stands on, provable with a clean plan and
nothing above it. `deed` connects to the operator's existing single AWS account
(under whatever org already owns it) via ambient SSO credentials — it does not
enable an Organization or create a member account; that machinery is deferred
until a real second project needs it (rule of three), and a flat single-account
config forecloses nothing. The Terraform state bucket for that account is
bootstrapped **by hand once** (make-bucket + versioning +
encryption + block-public-access) and thereafter only *referenced* by the
backend — never managed or imported by Terraform — with those imperative steps
documented in `deed`'s README so the one deliberately-imperative gesture is
repeatable. The configuration is flat in-repo HCL (no modules) wired to an S3
backend with native locking (`use_lockfile`) and scaffolded for the per-component
state split, with local `.tfstate` and the bootstrap-secret tfvars git-ignored so
state and secret values never land in the public repo. Apply is laptop-only under
short-lived IAM Identity Center (SSO) credentials; the root `justfile` grows
`deed` recipes and CI runs `fmt` / `validate` / `plan` and **never** `apply`.

**Blocked by:** None — can start immediately.

**Status:** in-review

- [x] `deed` connects to the operator's existing single account via ambient SSO creds; no Organization or member account is created — `deed/foundation/providers.tf`
- [x] State bucket hand-bootstrapped (versioning + encryption + block-public-access) and referenced by the backend, never managed or imported; steps documented in `deed`'s README — `deed/README.md`, `deed/foundation/versions.tf`
- [x] Flat in-repo HCL on an S3 backend with `use_lockfile`, scaffolded for the per-component state split (one root dir per component, own backend `key`)
- [x] `.tfstate` and the bootstrap-secret tfvars are git-ignored; no state and no secret values in the repo — root `.gitignore` + `deed/.gitignore`
- [x] Apply is SSO-only (no static IAM access key on disk); `just` deed recipes exist — `deed-fmt` / `deed-validate` / `deed-plan` / `deed-apply`
- [x] CI runs `fmt` / `validate` and never `apply` — `.github/workflows/deed.yml`. `plan` is kept local (`just deed-plan`) under SSO creds for now; CI does not touch the backend.

**Human verification still required:** a clean `plan` is only demonstrable after the state bucket is hand-bootstrapped and SSO creds are available; `terraform`/OpenTofu was not installed in the authoring env, so `fmt`/`validate` were verified by hand, not run.
