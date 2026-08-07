# ECR vs GHCR: pricing and auth for custodian image hosting

Context: custodian (Go service) built as a Docker image in GitHub Actions, pulled by one long-lived EC2 t4g.micro in eu-west-1. Deciding where to host the image.

Access date for all sources below: 2026-08-02.

## Amazon ECR pricing

- Private repo storage: **$0.10/GB-month** beyond free tier. ([AWS ECR Pricing](https://aws.amazon.com/ecr/pricing/))
- Same-region data transfer (EC2 <-> ECR within eu-west-1): **free ($0.00/GB)** — "Data transferred between Amazon ECR and other services within the same Region e.g., Amazon EC2 ... is free of charge." ([AWS ECR Pricing](https://aws.amazon.com/ecr/pricing/))
- Cross-region transfer out of private repos: $0.09/GB (not applicable here — CI push and EC2 pull are same-region).
- Data transfer *in* to private repos: free, always.
- ECR private free tier: 500 MB-month storage free for the first 12 months as a new AWS customer (not permanent). ([AWS ECR Pricing](https://aws.amazon.com/ecr/pricing/))
- ECR Public (brief): 50 GB/month storage always free for all accounts; 5 TB/month free egress for authenticated pulls, 500 GB/month free for anonymous pulls. Not relevant here since custodian images should stay private. ([AWS ECR Pricing](https://aws.amazon.com/ecr/pricing/))

## GHCR pricing

- GitHub Free personal account included quota: 500 MB storage + 1 GB data transfer/month, shared across all GitHub Packages usage on the account. ([About billing for GitHub Packages](https://docs.github.com/en/billing/managing-billing-for-github-packages/about-billing-for-github-packages))
- Public packages: storage and egress are **free**, uncapped by the quota above; data transferred in is always free regardless of visibility. ([About billing for GitHub Packages](https://docs.github.com/en/billing/managing-billing-for-github-packages/about-billing-for-github-packages))
- Private packages: consume the account's storage/transfer quota; once exhausted, overage is billed at roughly **$0.25/GB-month storage** and **$0.50/GB egress** (outside Actions workflows) per GitHub's published Packages rates. ([GitHub Packages billing (Enterprise Cloud docs, same rates apply)](https://docs.github.com/en/enterprise-cloud@latest/billing/managing-billing-for-your-products/about-billing-for-github-packages); rates corroborated via [GitHub Pricing Calculator](https://github.com/pricing/calculator))
- Caveat: GitHub currently does **not** meter/bill container registry (GHCR) storage or bandwidth at all — it's stated as "currently free," with GitHub reserving the right to start billing with one month's notice. This is a live point of community confusion/ambiguity, not a documented permanent guarantee. ([Container registry docs](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry); [community discussion on GHCR billing status](https://github.com/orgs/community/discussions/183054))

## Realistic monthly cost estimate

Workload: 1-3 images, each <100MB compressed, a handful of tags retained (say <1GB total storage), pulled a few times/day, all traffic same-region (EC2 in eu-west-1 pulling from ECR in eu-west-1, or EC2 pulling from GHCR over the public internet).

- **ECR**: ~1GB storage x $0.10/GB-month ≈ **$0.10/month** (~£0.08). Pull traffic is same-region EC2<->ECR and free regardless of volume. Total: **well under $0.50/month**, effectively ~$0.10/month.
- **GHCR**: If images stay private and usage stays under the free personal quota (500MB storage / 1GB transfer), cost is **$0/month** in practice today, since GHCR bandwidth/storage isn't currently metered at all per GitHub's own docs. If GitHub does start billing per the standard Packages rates, a handful of daily pulls of <100MB images could push data transfer over 1GB/month (e.g. 3 images x 100MB x 3 pulls/day x 30 days ≈ 27GB/month), which at $0.50/GB would be **~$13-15/month** — an order of magnitude more expensive than ECR for the exact same workload.

Bottom line: ECR is cheaper and its cost is fixed/predictable (near-zero, same-region transfer is free by design). GHCR is free *today* only because GitHub hasn't turned on registry billing yet; if it does, this workload's pull volume would land squarely in paid egress territory.

## Auth/status

- **ECR**: confirmed — an EC2 instance profile (IAM role) supplies temporary, auto-rotating credentials via the instance metadata service; `aws ecr get-login-password` uses these to mint a Docker login token. No static AWS credentials are ever stored on the box. ([Amazon ECR private registry authentication](https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry-auth.html))
- **GHCR**: confirmed — GitHub Packages authentication outside of a GitHub Actions workflow (i.e. a `docker login`/`docker pull` from a persistent EC2 box) only supports a personal access token (classic) with `read:packages` scope; there is no OIDC or short-lived/instance-role equivalent for plain `docker pull`. That token would need to live on the box indefinitely (or be rotated manually), and its scope is broader than "pull this one registry" (classic PATs are account-wide-scoped, not registry/repo-scoped). ([Working with the Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry))

## Verdict

The ECR lean is confirmed. ECR is materially cheaper for this workload (near-zero, same-region transfer is free by design) and its pricing is stable/documented, whereas GHCR is only free because container registry billing isn't switched on yet — a policy that could change with one month's notice. On auth, ECR's IAM-instance-profile model means zero long-lived secrets on the box, while GHCR would require storing a broad-scoped, long-lived PAT there indefinitely. Go with ECR.
