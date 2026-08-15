# 01 — Terraform foundation: Org member account, S3 state backend, CI plan gate

**What to build:** The ground `deed` stands on, provable with a clean plan and
nothing above it. AWS Organizations is enabled with the playground living in its
own member account, so a future second project becomes a new account rather than
a cross-account migration of stateful resources. The Terraform state bucket for
that account is bootstrapped **by hand once** (make-bucket + versioning +
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

**Status:** ready-for-agent

- [ ] AWS Organizations enabled; the playground runs in its own member account
- [ ] State bucket hand-bootstrapped (versioning + encryption + block-public-access) and referenced by the backend, never managed or imported; steps documented in `deed`'s README
- [ ] Flat in-repo HCL on an S3 backend with `use_lockfile`, scaffolded for the per-component state split
- [ ] `.tfstate` and the bootstrap-secret tfvars are git-ignored; no state and no secret values in the repo
- [ ] Apply is SSO-only (no static IAM access key on disk); `just` deed recipes exist
- [ ] CI runs `fmt` / `validate` / `plan` against the real backend and never `apply`; a clean plan is demonstrable
