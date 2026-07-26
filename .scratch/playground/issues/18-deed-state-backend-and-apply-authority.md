# deed: state backend and apply authority

Type: grilling
Status: open

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
