# Research: AWS pricing for mitigating a CloudFront "wallet DoS" on a small personal site

**Date**: 2026-07-31
**Question**: What does CloudFront, WAF, and AWS Budgets actually cost (eu-west-2 / Europe pricing), and what's the cheapest combination that protects a ~£10/month personal-site CloudFront distribution against someone hammering cached static assets to run up the bill?

## Bottom line

- **CloudFront's perpetual free tier (1 TB egress + 10M requests/month, no 12-month limit) does not save you from a real attack.** A 100M-request / ~20 TB month blows through it and produces a bill of roughly **$1,678** (~£1,320) for that month alone — see worked arithmetic below.
- **AWS Budgets can only alert, not stop or throttle CloudFront spend.** The pricing page's own wording confirms actions are limited to changing "IAM and Service Control Policy permissions... and AWS resources when thresholds are exceeded" — there is no documented CloudFront-throttle action, and monitoring/alerts are explicitly after-the-fact ("receive notifications on your budgets").
- **A WAF Web ACL + one rate-based rule costs a fixed $6/month floor** ($5 Web ACL + $1 rule) regardless of traffic, plus $0.60 per million requests it *processes* (not just blocks) — under the 100M-request attack scenario that's an extra $60, i.e. $66/month for the WAF layer alone, which is proactive (can block abusive IPs before they keep pulling data) rather than reactive.
- Budgets alone is nearly free ($0/month steady-state, since notification-only budgets are free and only *action-enabled* budgets have a cost, with the first two of those also free) but is a **smoke detector, not a sprinkler system**: it tells you the bill is climbing; it does not reduce it.

## 1. CloudFront pricing (Europe)

- **HTTPS requests, Europe (incl. Israel/Türkiye group)**: **$0.0120 per 10,000 requests** (i.e. $0.0000012/request). Confirmed via web search cross-referencing AWS's pricing page content (WebFetch on `aws.amazon.com/cloudfront/pricing/` itself returned a flat-rate-plan page layout, not the on-demand region table, so this figure is sourced from search results quoting the AWS on-demand table directly, cross-checked against two independent third-party pricing trackers that both cite $0.0120/10K for Europe).
- **Data transfer out to internet, Europe, tiered**:
  - First 10 TB/month: **$0.085/GB**
  - Next 40 TB/month: **$0.080/GB**
  - Next 100 TB/month: **$0.060/GB**
  - (further tiers exist below $0.060 at higher volumes, not relevant here)
- **Perpetual (always-free, not 12-month) free tier**: confirmed directly via WebFetch of the official AWS News Blog post announcing the expansion — exact quotes: *"Data Transfer from Amazon CloudFront is now free for up to 1 TB of data per month"*, *"raising the number of free HTTP and HTTPS requests from 2,000,000 to 10,000,000"*, and *"is no longer limited to the first 12 months after signup"*. So: **1 TB egress/month + 10,000,000 HTTP/HTTPS requests/month, forever, no 12-month cutoff.**
- **Caveat on sourcing**: direct WebFetch of `https://aws.amazon.com/cloudfront/pricing/` did not render the on-demand region-rate table (it returned what looks like a newer flat-rate-plan marketing layout — Free/Pro/Business/Premium tiers at $0/$15/$200/$1,000/month with bundled allowances — which may be a redesigned page or a fetch artifact). The per-GB and per-10K-request figures above come from WebSearch results that explicitly quote AWS's on-demand pricing table text, corroborated by two independent CDN-cost-tracking sites (egresscost.com, blazingcdn.com) giving matching numbers. The free-tier figures **are** directly confirmed from an official AWS blog post fetch. Treat the on-demand unit prices as "AWS-sourced via search, not directly re-rendered by WebFetch" rather than a first-party page fetch.

## 2. AWS WAF pricing

Sourced via WebFetch of `https://aws.amazon.com/waf/pricing/`:

- **Web ACL**: **$5.00/month** (charged per Web ACL you create)
- **Rule**: **$1.00/month per rule** created by you, per Web ACL
- **Rate-based rule**: the pricing page does **not** list a separate price for rate-based rules — they are billed the same as any other rule, **$1.00/month**
- **Requests processed**: **$0.60 per million requests** processed by WAF (charged regardless of whether a rule matches/blocks)
- Fees are prorated hourly; managed rule groups from AWS Marketplace vendors carry additional seller-set fees (not relevant here — a single self-authored rate-based rule doesn't need one)

**Fixed monthly floor, one rate-based rule, negligible traffic:**
```
Web ACL           $5.00
1 rule            $1.00
------------------------
Fixed floor       $6.00/month
```

**Same Web ACL under a 100,000,000-request attack month:**
```
Fixed floor                          $ 6.00
Request processing: 100M / 1M = 100 units
  100 × $0.60                      $ 60.00
------------------------------------------
Total WAF cost, attack month        $66.00
```

## 3. AWS Budgets pricing

Sourced via WebFetch of `https://aws.amazon.com/aws-cost-management/aws-budgets/pricing/`:

- **Plain budget monitoring/alerts**: free. Exact quote: *"You can monitor and receive notifications on your budgets free of charge."*
- **Action-enabled budgets**: *"Your first two action-enabled budgets are free (regardless of the number of actions you configure per budget) per month. Afterwards each subsequent action-enabled budget will incur a $0.10 daily cost."* — so a 3rd+ action-enabled budget costs $0.10/day (~$3/month), but for a single personal site you'd need at most one or two, which are free.
- **Budget Reports**: *"Each report delivered will incur a cost of $0.01"* — a separate, optional feature (scheduled cost report emails), not required for alerting.
- **Can Budgets automatically stop/throttle spend?** No. The page's own description of "Budgets Actions" scopes them to *"control IAM and Service Control Policy permissions as well as AWS resources when thresholds are exceeded"* — i.e. an action can revoke IAM permissions or apply an SCP (e.g. to stop a *human or process from spending more*, such as detaching a role's ability to launch instances), but there is no CloudFront-specific "throttle requests" or "stop serving" action. Nothing on the page describes automatically reducing or capping CloudFront's request/data-transfer billing once traffic is already hitting the distribution — Budgets Actions can lock down *account permissions*, not CDN traffic. For a wallet-DoS via legitimate cached-asset requests, there is no Budgets mechanism that intercepts the requests themselves.

## 4. Worked scenario: unmitigated 100M-request / ~20 TB attack month

Inputs: 100,000,000 cached HTTPS requests, ~200 KB average object size ⇒ 100,000,000 × 200 KB = 20,000,000,000 KB = 20,000,000 MB = **20,000 GB (20 TB)** total egress.

**Request cost:**
```
Total requests                 100,000,000
Free tier (perpetual)          -10,000,000
Billable requests               90,000,000
90,000,000 / 10,000 = 9,000 billable units
9,000 × $0.0120                    = $108.00
```

**Egress cost:**
```
Total egress                    20,000 GB
Free tier (perpetual)           -1,000 GB
Billable egress                 19,000 GB

Tier 1 — first 10 TB (10,000 GB) of the billable amount @ $0.085/GB:
  10,000 × $0.085                = $850.00
Tier 2 — remaining 9,000 GB @ $0.080/GB (next-40TB tier):
  9,000  × $0.080                = $720.00
Egress subtotal                  = $850.00 + $720.00 = $1,570.00
```

**Total CloudFront bill for that month:**
```
Requests   $108.00
Egress   $1,570.00
--------------------
TOTAL    $1,678.00   (≈ £1,320 at ~0.79 USD/GBP — illustrative only, not a cited FX rate)
```

This is the number a purely reactive control (Budgets alert) would report to you *after* it had already been incurred — it cannot prevent it.

## 5. Recommendation: Budgets-only vs WAF + Budgets

| | **Option A: Budgets alarm only** | **Option B: WAF Web ACL + 1 rate-based rule + Budgets alarm** |
|---|---|---|
| **Steady-state fixed cost (near-zero traffic)** | **$0.00/month** — plain notification budgets are free; a single alert-only budget needs no action-enabled tier | **$6.00/month** ($5 Web ACL + $1 rule) — Budgets alert itself still free |
| **Cost under the 100M-request attack month** | $0 extra for the alarm itself (it only notifies) — full $1,678 CloudFront bill still lands | $66 for WAF (fixed $6 + $60 request processing) **plus** whatever residual CloudFront egress/requests get through before the rate-based rule throttles the source IP(s) — materially less than $1,678 if the rule catches the abusive IPs early, though CloudFront may still bill for requests WAF evaluates before blocking (not confirmed either way in AWS's pricing docs — treat as an open question, not a claim) |
| **What it protects against** | Nothing proactively. Tells you, via email/SNS, that spend crossed a threshold — useful as a trip-wire so a human notices and manually intervenes (e.g., disable the distribution, add WAF after the fact) | A rate-based rule can automatically block/rate-limit an individual IP (or IP set) once it exceeds a request-count threshold in a rolling window, stopping the attacker's requests **at the edge before CloudFront serves the cached object again**, i.e. before further egress is generated from that source — this is the only one of the two that can act *during* the attack |
| **What it does NOT do** | Cannot stop or slow CloudFront from serving requests; cannot revoke access; the bill keeps growing until you personally act | Does not stop a *distributed* attack from many unique IPs as cleanly (a single-IP rate-based rule is weakest against low-and-slow botnets); does not eliminate WAF's own $0.60/million processing cost, which itself scales with attack volume; still needs a human/process to actually create and tune the rule's rate threshold |
| **Judgement for a £10/month personal site** | Cheap safety-net but not a mitigation — pair with manual runbook (e.g., know how to flip the distribution off or add WAF quickly) | The $6/month fixed floor comfortably fits inside a ~£10/month total budget alongside a t4g.micro EC2 instance and low-traffic S3/CloudFront usage, and is the only option that can actually cap the damage of a wallet-DoS in progress rather than just reporting it afterward |

## Sources & fetch date (2026-07-31)

- `https://aws.amazon.com/cloudfront/pricing/` — fetched via WebFetch; returned a flat-rate-plan page layout ($0/$15/$200/$1,000 tiers), not the on-demand per-region table. Free-tier and on-demand unit-price figures for this doc were instead corroborated via WebSearch results quoting AWS's on-demand table text, and via a direct WebFetch of the official AWS News Blog post below.
- `https://aws.amazon.com/blogs/aws/aws-free-tier-data-transfer-expansion-100-gb-from-regions-and-1-tb-from-amazon-cloudfront-per-month` — fetched via WebFetch; source for the perpetual 1 TB / 10M-request CloudFront free tier quotes.
- `https://aws.amazon.com/waf/pricing/` — fetched via WebFetch; source for Web ACL ($5/month), rule ($1/month), and per-million-requests ($0.60) figures.
- `https://aws.amazon.com/aws-cost-management/aws-budgets/pricing/` — fetched via WebFetch (twice, to confirm exact wording); source for free notification budgets, the "first two action-enabled budgets free" rule, $0.10/day for subsequent action-enabled budgets, $0.01/report, and the scope of Budgets Actions (IAM/SCP/resource controls, not CloudFront throttling).
- WebSearch queries used to cross-check/locate the Europe on-demand CloudFront rates ($0.0120/10K HTTPS requests; $0.085 / $0.080 / $0.060 per-GB tiers), corroborated across egresscost.com and blazingcdn.com third-party pricing trackers, both dated 2026.
