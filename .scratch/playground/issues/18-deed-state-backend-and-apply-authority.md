# deed: state backend and apply authority

Type: grilling
Status: resolved

## Question

Where does `deed`'s state live, how is it locked, and who is allowed to `apply`?

- **The tool itself** — Terraform (BUSL-licensed since 1.6) or OpenTofu (MPL, Linux Foundation fork)? Affects nothing else in this ticket, but it wants deciding before any HCL is written.
- **Backend** — S3 with native state locking (`use_lockfile`, added in Terraform 1.10, no DynamoDB table needed), the older S3 + DynamoDB lock table, or a hosted backend's free tier?
- **Bootstrap** — the state bucket is itself infrastructure. Created by hand once and adopted, or does `deed` chicken-and-egg its own backend?
- **Apply authority** — do you `apply` from your laptop under an IAM user or SSO role, or does CI apply via GitHub OIDC with no long-lived keys?
- **Blast radius** — one state file for everything, or split per component? `20` pushes on this again once non-playground projects arrive.
- **Drift** — is `plan` run on a schedule to catch changes made by hand in the console, or is drift simply accepted?

## Context

Surfaced while resolving `01`. That ticket committed to EC2, S3, CloudFront, ACM, Route 53 and an IAM instance profile — enough real infrastructure that it wants declaring rather than clicking, which is what made `deed` a component in the first place.

`01`'s credential principle should carry through: custodian deliberately holds no long-lived AWS credentials, so `deed` adopting a long-lived access key for CI would be inconsistent with the decision that justified choosing AWS at all. GitHub OIDC is the analogous answer.

Note that `deed` is the one component capable of destroying every other one. Apply authority is a security decision here, not merely a convenience one.

## Blocks

`20` deed multi-project layout

## Answer

- **Tool: Terraform** (not OpenTofu). Chosen for career-alignment — the industry-default HCL toolchain, mirroring `07`'s "career-aligned learning" logic for Go. BUSL doesn't bite at this scale; OpenTofu's drop-in compatibility means the skill transfers if ever wanted.
- **Backend: S3 with native locking** (`use_lockfile = true`, Terraform 1.10+). State lives in S3 — the rented-durability layer already used by `08`/`11` — and the `.tflock` conditional-write object replaces the legacy DynamoDB lock table (one fewer moving part, same instinct that killed Redis in `08` and CloudWatch in `11`). HCP/hosted backend rejected: reintroduces a SaaS dependency and long-lived-credential surface against `01`.
- **State never committed to git.** Terraform state is plaintext secrets; committing it to this repo's *clean public history* (`13`) would be the `NEXT_PUBLIC_NOTION_KEY` mistake in a different file. It's also unmergeable JSON and exactly the "if it's gone, it's gone" data `01` says to rent, not self-manage on a mortal box. Local `.tfstate` is `.gitignore`'d; S3 versioning + encryption-at-rest hold the real copy.
- **Bootstrap: create the state bucket by hand once, then reference it — never manage it.** ~4 CLI commands (mb + versioning + encryption + block-public-access), documented in `deed`'s README, left *outside* Terraform's reach. Explicitly **not** `terraform import`ed: a managed state bucket is a footgun (a botched refactor/destroy could delete the bucket holding its own state). This is the one place clicking beats declaring; the self-management-teaches-you ethos is still satisfied.
- **Apply authority: laptop-only, under short-lived IAM Identity Center (SSO) creds.** No static IAM access key on disk for the one identity that can destroy everything — the sharpest case of `01`'s "nothing long-lived on disk" principle. **CI never applies** — deed's CI stays capped at `fmt`/`validate`/`plan` per `13`. GitHub OIDC parked as fog for a future gated CI-apply (`13` already flags this).
- **Blast radius: multiple state files, split by logical component** (e.g. static-site, custodian, shared edge/DNS) — not one monolith. *Exact* boundaries and where the shared CloudFront/Route 53/ACM land are deliberately left to implementation time (not worth pre-deciding). The one invariant kept regardless of layout: **`lifecycle { prevent_destroy = true }` on the data buckets** (SQLite backups + media) so no component `destroy` can silently take unrecoverable data. `20` inherits the principle: state boundary = logical grouping; new projects get their own state(s).
- **Drift: accepted, personal responsibility.** No scheduled `plan` — that would need standing read-only AWS creds in CI, reintroducing exactly the credential surface Question 4 eliminated, to solve a multi-operator problem a solo playground doesn't have. Drift surfaces for free at the next local `apply`. Scheduled read-only plan parked as fog, riding the same OIDC path if it ever graduates.

**Unblocks `20`.** Hands `20` the principle (state boundary = logical component; shared cross-cutting resources get their own state) and the `prevent_destroy` data-bucket invariant.
