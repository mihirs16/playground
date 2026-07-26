# Hosting and deployment posture

Type: grilling
Status: resolved

## Question

Where does each component actually run, and where does uploaded media live?

Settle three things together, because they trade off against each other:

1. **custodian's runtime** — one always-on process (Fly.io / Railway / Hetzner), serverless functions (Cloudflare Workers, Vercel), or self-hosted on your own box?
2. **Media storage** — object storage (Cloudflare R2, S3) or a disk you own?
3. **persona's host** — static hosting (Cloudflare Pages, Netlify, Vercel) and whether it needs any server-side runtime at all.

## Context

Explicitly deferred during the charting session — you were asked and chose to make it a ticket rather than decide on the spot.

This is the map's biggest constraint-setter, which is why it blocks so much. A long-lived process can buffer multipart uploads, hold connections, and cache Steam responses in memory for free; a serverless custodian has to push every one of those into an external service, and that changes both its language options and its storage model.

Prior recommendation on offer, for what it's worth: long-lived container plus R2 (zero egress, S3-compatible so not a lock-in), persona static on Pages. The argument is that custodian stays boring and debuggable — which matters more than usual now that blogs load from it at runtime, making it load-bearing for the site rather than merely convenient.

Cost matters here and is a legitimate input — this is a personal project, not a funded one.

## Blocks

`07` custodian language, `08` storage model, `11` observability

## Answer

**Governing principle: self-manage the compute, rent the durability.** Self-manage things whose failure mode is "it's down for an hour and you learn something"; rent things whose failure mode is "the data is gone." The playground exists to learn by administering things, so a hand-run box is on-mission — but nothing irreplaceable should depend on a backup routine written by hand.

Everything runs in **AWS, eu-west-1 (Ireland)**, on an account that already exists with IAM configured.

### custodian

- **Runtime**: one always-on **EC2 `t4g.micro`** — 2 vCPU Graviton (ARM64), 1 GB RAM, Ubuntu. nginx as reverse proxy, a systemd unit, hand-administered on purpose.
- **On-demand pricing**, no Savings Plan until the shape settles. Resizing to `t4g.small` is a stop / change type / start — roughly two minutes, no rebuild — so 1 GB is a reversible bet rather than a commitment.
- **State**: **SQLite in-process**, on a 20 GB gp3 EBS volume, **continuously replicated to S3** (Litestream or equivalent). No second daemon to administer, and no managed database on the critical path of every blog page load.
- **Credentials**: an **IAM instance profile**. custodian stores no long-lived AWS credentials — the SDK picks up auto-rotating short-lived ones from instance metadata.

### Media and blobs

**S3**, eu-west-1. The split is **blobs vs records**, not "blogs and media vs everything else":

- **Blobs** — uploaded images, the resume PDF. Large, opaque, served by URL, never queried. S3.
- **Records** — blog title, slug, tags, publish date, draft flag, ordering, the markdown body itself, and the derived Steam/GitHub cache with its fetch timestamp. Small, and queried. SQLite.

Blog markdown is a record, not a blob. Putting it in S3 means hand-rolling an index object to answer "published posts, newest first", and racing against that index on every write — a database with extra steps and no transactions.

### persona

Private **S3 bucket + CloudFront** via origin access control. **Leaving Netlify.**

### Edge topology

**One CloudFront distribution fronts both persona and custodian** — the S3 bucket as static origin, the EC2 box as a second origin. This buys three things at once:

- **Edge caching for blog responses**, which is what makes the origin's physical location a non-issue.
- **Same-origin** — no CORS preflights, and cookie-based auth stays available to `10` rather than being ruled out on day one.
- **One ACM certificate**, auto-renewing and free.

The URL shape — path prefix versus subdomain — belongs to `09`.

**SSR is not foreclosed.** `15` and `16` haven't decided whether persona needs server-side rendering for blog crawlability. CloudFront fronts any origin, so if it turns out to, an origin pointing at the same box absorbs it.

### DNS and TLS

- Registration stays at **Squarespace**; nameservers delegate to **Route 53** (~$0.50/mo). Not optional — apex domains can't CNAME, and Route 53 alias records are the clean way to point `mihirsingh.dev` at CloudFront.
- **ACM certificates for CloudFront must be issued in us-east-1**, regardless of where everything else lives.
- The *sequencing* of the cutover from Netlify stays in the map's fog.

### Cost

~**£10/month** at July 2026 prices:

| Line | Monthly |
|---|---|
| EC2 `t4g.micro`, eu-west-1 | £5.20 |
| Public IPv4 address | £2.90 |
| EBS gp3, 20 GB | £1.25 |
| S3 | £0.25 |
| CloudFront — permanent free tier: 1 TB out, 10M requests | £0 |
| Route 53 hosted zone | £0.40 |
| **Total** | **~£10.00** |

Egress is free to 100 GB/month across all of AWS, which this will never approach. **Set an AWS Budgets alert at £20/month before provisioning anything** — AWS can surprise you on the bill in a way a flat-fee VPS cannot.

### Rejected, and why

- **Hetzner CX22 + Cloudflare R2** (~£4/mo — this ticket's prior recommendation). Genuinely cheaper, and identical on the self-managed-box learning. Rejected because R2 requires a long-lived access key on disk, a milder version of the `NEXT_PUBLIC_NOTION_KEY` mistake this project exists to avoid repeating. IAM instance profiles remove the secret entirely. The ~£6/mo premium also buys AWS fluency, which is worth more than Hetzner fluency.
- **Serverless custodian.** Never seriously in play once a long-lived process was wanted for SQLite, caching and upload buffering.
- **Postgres on the box.** The worst square of the grid: you take on the operational surface *and* personally own the irreplaceable data.
- **Managed Postgres** (Neon, Supabase free tier). Rents the thing worth self-managing, and puts a free-tier dependency — with idle-to-sleep behaviour — on the critical path of every blog page load.
- **AWS Lightsail.** Looks like the cheap simple option, but Lightsail instances can't cleanly assume IAM roles, which throws away the entire reason for choosing AWS.
- **Staying on Netlify for persona.** Free and already wired, but a second vendor, and it forfeits the single-distribution benefits above.
- **eu-west-2 (London).** Roughly £1/mo more for ~10ms nobody will perceive.

### Constraints handed to blocked tickets

- **`07` language and runtime** — must run comfortably in **1 GB on ARM64**, embed **SQLite in-process**, and use an AWS SDK supporting the instance-metadata credential chain. A long-lived process is assumed. Every serious candidate satisfies all three, so this narrows the field less than it might look — but "boring and debuggable" gained weight, since the box is now personally administered.
- **`08` storage model** — the *posture* is locked: state on the box, durability rented to S3, blobs in S3. The **engine** is still `08`'s to confirm, and the git-backed-markdown fork is **narrowed but not dead** — a hybrid, markdown in git with an index in SQLite, survives this. Media is settled: S3.
- **`10` auth model** — same-origin means **cookies are available**. No long-lived AWS credentials exist to protect. Third-party API secrets still need a home; `19` asks where.
- **`11` observability** — a box you administer, so no managed APM arrives by default. CloudWatch is in reach and coherent with the AWS choice. External uptime probing gains value, since a single instance is a single point of failure.
- **`13` monorepo layout** — must place `deed`.

### Scope change: a fifth component

`deed` — the Terraform component — was surfaced by this ticket, since the decisions above amount to enough infrastructure that it wants declaring rather than clicking. It provisions the playground **and projects beyond it**, which gives it the same property `blank` has: its interface must not become playground-shaped.

The destination is redrawn from four components to five. Three tickets follow: `18` state backend and apply authority, `19` provisioning boundary, `20` multi-project layout.

Full rationale is also recorded as [ADR-0001](../../../docs/adr/0001-hosting-and-deployment-posture.md).
