# ADR-0001: Hosting and deployment posture

**Status**: Accepted
**Date**: 2026-07-25
**Deciding ticket**: [`01` Hosting and deployment posture](../../.scratch/playground/issues/01-hosting-and-deployment-posture.md)

## Context

The playground needs somewhere to run. Three questions trade off against each other and had to be settled together: custodian's runtime, where uploaded media lives, and persona's host.

Three facts constrained the decision.

**custodian is load-bearing at runtime.** persona bakes profile content at build time but fetches blogs from custodian on every page view. custodian being down is the site's primary content being down — which is why a long-lived, debuggable process was wanted over a serverless one, and why monitoring is a stated requirement rather than a nice-to-have.

**Credentials are the known failure mode.** The deprecated site put its Notion token in `NEXT_PUBLIC_NOTION_KEY`, inlining a secret into the client bundle. custodian holds all third-party credentials, so any hosting choice that requires long-lived secrets sitting on disk is a milder repeat of the same mistake.

**The playground exists to learn.** Self-administering infrastructure is on-mission, not a cost dodge. But that cuts both ways: some operational work teaches something durable, and some is toil where failure is unrecoverable.

An AWS account with IAM already configured was available. The owner is UK-based and cost-conscious.

## Decision

### Governing principle

**Self-manage the compute, rent the durability.** Self-manage things whose failure mode is "it's down for an hour and you learn something." Rent things whose failure mode is "the data is gone."

### Placement

Everything runs in **AWS, eu-west-2 (London)**.

| Concern | Decision |
|---|---|
| custodian runtime | One always-on **EC2 `t4g.micro`** — 2 vCPU Graviton (ARM64), 1 GB, Ubuntu. nginx reverse proxy, systemd unit, hand-administered. On-demand pricing |
| custodian state | **SQLite in-process** on a 20 GB gp3 EBS volume, continuously replicated to S3 |
| Media and blobs | **S3**, eu-west-2 |
| Credentials | **IAM instance profile** — no long-lived AWS credentials anywhere |
| persona host | Private **S3 bucket + CloudFront** via origin access control |
| Edge | **One CloudFront distribution fronting both** persona (S3 origin) and custodian (EC2 origin) |
| TLS | **ACM**, issued in us-east-1 as CloudFront requires |
| DNS | Registration stays at **Squarespace**; nameservers delegate to **Route 53** |

### Blobs versus records

Storage splits on **shape, not on content type**:

- **Blobs** — uploaded images, the resume PDF. Large, opaque, served by URL, never queried. S3.
- **Records** — blog title, slug, tags, publish date, draft flag, ordering, the markdown body itself, and the derived Steam/GitHub cache with its fetch timestamp. Small, and queried. SQLite.

Blog markdown is a record. Listing an S3 bucket returns keys, not "published posts, newest first" — so an S3-only content store means maintaining a separate index object and racing against it on every write. That is a database with extra steps and no transactions.

### Cost

~£10/month at July 2026 prices: EC2 £5.20, public IPv4 £2.90, EBS £1.25, S3 £0.25, CloudFront £0 (permanent free tier — 1 TB egress, 10M requests), Route 53 £0.40. Egress is free to 100 GB/month across all of AWS.

An **AWS Budgets alert at £20/month** is a precondition of provisioning anything.

## Alternatives considered

**Hetzner CX22 + Cloudflare R2, ~£4/month.** The closest call, and this ADR's most contestable decision. Cheaper by ~£6/month, and identical on the self-managed-box learning — an EC2 instance is just a VPS. Rejected because R2 requires a long-lived access key written to disk on the box. IAM instance profiles remove the secret entirely, which is a categorically better answer to the failure mode this project exists to avoid repeating. The premium also buys AWS fluency, which transfers further than Hetzner fluency.

**Serverless custodian.** Off the table once a long-lived process was wanted for in-process SQLite, caching, and buffering multipart uploads.

**Postgres on the box.** The worst combination available: you take on the operational surface of a second daemon *and* personally own the irreplaceable data — violating the governing principle on both sides at once.

**Managed Postgres (Neon, Supabase free tier).** Rents the thing worth self-managing, and puts a free-tier dependency with idle-to-sleep behaviour on the critical path of every blog page load.

**AWS Lightsail.** Appears to be the cheap simple option, but Lightsail instances cannot cleanly assume IAM roles, which discards the entire reason for choosing AWS.

**Keeping persona on Netlify.** Free and already wired. Rejected for a second vendor and for forfeiting the single-distribution benefits below.

**eu-west-1 (Ireland).** Roughly £1/month cheaper, for ~10ms nobody will perceive.

## Consequences

### Good

- **No long-lived AWS credentials exist**, so there is nothing to rotate, leak, or accidentally commit.
- **One CloudFront distribution over both components** means blog responses cache at the edge (making origin location a non-issue), the API is same-origin (no CORS preflights, and cookie auth stays available), and one auto-renewing certificate covers everything.
- **One vendor, one bill, one IAM, one console** — and real, transferable AWS learning in ACM, OAC, cache behaviours and instance profiles.
- **SSR is not foreclosed.** CloudFront fronts any origin, so if persona later needs server-side rendering for blog crawlability, an origin pointing at the same box absorbs it.
- **Resizing is cheap.** `t4g.micro` → `t4g.small` is a stop / change type / start, roughly two minutes with no rebuild.

### Bad, or merely accepted

- **~£6/month more than the cheapest viable build.** Bought deliberately, for the credential story and the AWS exposure.
- **The public IPv4 charge is ~30% of the instance bill** — an AWS-specific tax with no Hetzner equivalent.
- **Cache invalidation on deploy** is now yours. Netlify did it for free; 1,000 invalidation paths/month are free, then $0.005 each.
- **ACM certs for CloudFront must live in us-east-1** while everything else is in eu-west-2. A well-known gotcha that will bite exactly once.
- **A single instance is a single point of failure.** No redundancy, and blogs load from it at runtime. This raises the value of *external* uptime probing — a check running on the box cannot report that the box is gone.
- **Concurrent SQLite writes serialise.** Irrelevant at one author publishing a few times a month; it would matter if this were ever multi-tenant.
- **State on the instance's disk pushes against immutable infrastructure.** Replacing the instance means moving or re-restoring the volume. Continuous replication to S3 makes it survivable, but the deploy model has to say so explicitly.
- **AWS can surprise you on the bill** in a way a flat-fee VPS cannot. Hence the budget alarm.

### Downstream constraints

- **custodian's language** must run comfortably in 1 GB on ARM64, embed SQLite in-process, and use an AWS SDK supporting the instance-metadata credential chain. Every serious candidate qualifies — but the box is hand-administered, so "boring and debuggable" now carries more weight.
- **The storage model's posture is locked** (state on the box, durability rented to S3, blobs in S3); the engine is not. A git-backed hybrid — markdown in git, index in SQLite — survives this decision.
- **The auth model** gains cookies via same-origin, and loses the need to manage AWS credentials at all. Third-party API secrets still need a home.
- **Observability** happens on a box you own. No managed APM by default; CloudWatch is in reach.

### A fifth component

The infrastructure above amounts to enough moving parts that it wants declaring rather than clicking. That established **`deed`**, the infrastructure-as-code component, redrawing the effort's destination from four components to five.
