# deed: provisioning boundary

Type: grilling
Status: open
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
