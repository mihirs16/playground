# 03 — ECR repository, lifecycle policy & instance-profile pull grant

**What to build:** The registry custodian's image lives in and the box pulls from
with zero static credentials. `deed` provisions an ECR repository, a lifecycle
policy (keep last N / expire untagged) so image storage stays flat and cheap as
versions accumulate, and an ECR pull grant on the box's instance profile so the
box authenticates to the registry via the instance profile alone — no long-lived
registry PAT on the box (the reason GHCR was rejected). With this and 02 in place,
a custodian image pushed to ECR can be pulled and run on the box.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] ECR repository provisioned
- [ ] Lifecycle policy keeps last N and expires untagged images
- [ ] Instance profile carries an ECR pull grant; no long-lived registry credential exists on the box
