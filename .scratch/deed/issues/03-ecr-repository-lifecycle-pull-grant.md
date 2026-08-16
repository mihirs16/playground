# 03 — ECR repository, lifecycle policy & instance-profile pull grant

**What to build:** The registry custodian's image lives in and the box pulls from
with zero static credentials. `deed` provisions an ECR repository, a lifecycle
policy (keep last N / expire untagged) so image storage stays flat and cheap as
versions accumulate, and an ECR pull grant on the box's instance profile so the
box authenticates to the registry via the instance profile alone — no long-lived
registry PAT on the box (the reason GHCR was rejected). With this and 02 in place,
a custodian image pushed to ECR can be pulled and run on the box.

**Blocked by:** 02.

**Status:** ready-for-human

- [x] ECR repository provisioned — `deed/compute/registry.tf` (`aws_ecr_repository.custodian`), name in `deed/compute/variables.tf` (`ecr_repository_name`)
- [x] Lifecycle policy keeps last N and expires untagged images — `deed/compute/registry.tf` (`aws_ecr_lifecycle_policy.custodian`: rule 1 expires untagged after 1 day, rule 2 keeps the most recent `ecr_image_retention_count` images)
- [x] Instance profile carries an ECR pull grant; no long-lived registry credential exists on the box — `deed/compute/registry.tf` (`aws_iam_role_policy.ecr_pull` on `aws_iam_role.box`: repo-scoped layer/image reads + registry-level `ecr:GetAuthorizationToken`; the box authenticates via the instance profile, no PAT)

Implemented and code-reviewed on the `deed` branch (commit `deed(03)`); `terraform fmt`/`validate` pass. **Awaiting human apply** under SSO credentials (`plan`/`apply` are laptop-only), then verify a pushed image pulls on the box via the instance profile — mirroring the human-verified gate on ticket 02.
