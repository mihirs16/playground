# deed: multi-project layout

Type: grilling
Status: resolved
Blocked by: 18

## Question

How is `deed` structured so it can provision projects beyond the playground without becoming playground-shaped?

- **Module boundaries** — what is a reusable module (a "static site behind CloudFront", a "small always-on box") versus a playground-specific root configuration?
- **State separation** — one state file per project, per environment, or per component? Follows from `18`'s backend choice, but is a distinct question about blast radius *across* projects.
- **Where other projects live** — do they get root configurations inside `deed`, or do they consume `deed`'s modules from their own repos? The first keeps everything in one place; the second makes `deed` a published dependency, which is the `blank` pattern.
- **AWS account structure** — one account for everything, or Organizations with an account per project? Cheap to establish early, painful to retrofit.
- **Versioning** — if other projects consume `deed`'s modules, do they pin a version? `blank` publishes with semver for exactly this reason.

## Context

Surfaced while resolving `01`, when `deed` was established as the fifth component. It will own deployments for projects outside the playground — which gives it the same property `blank` has: being consumed from outside is the forcing function that stops the interface silently becoming playground-shaped.

Blocked on `18` because module and project boundaries are meaningless until state boundaries are settled.

Note the scope line. Making `deed` *capable* of provisioning other projects is in scope. Actually provisioning them is not — mirroring the existing out-of-scope ruling on adopting `blank` elsewhere.

## Answer

`deed` gets the **minimum structure that keeps it from going playground-shaped without building any multi-project machinery today** — one deliberate investment (Organizations), everything else deferred.

- **Consumption model — single in-repo config.** `deed` is one repo holding all the Terraform needed now. Published-module distribution (the `blank` "consumed from another repo" pattern) is **fog**, graduating only when a real second repo needs it. The forcing function against playground-shaping is a clean module interface, not the act of publishing — so publishing can wait.
- **Modules — none yet; flat now, extract on real reuse.** Straightforward playground HCL (`ec2`/`cloudfront`/`s3`/…), **no `modules/`↔`projects/` indirection**. "Capable of provisioning other projects" is satisfied because flat HCL forecloses nothing — extraction into a module is mechanical. Deferred to the first *real* second consumer (rule of three), so the interface is designed against two concrete examples, not one imagined one (a one-consumer interface just re-encodes that consumer). Consistent with the `08`/`11`/`18` "one fewer moving part" instinct.
- **Versioning — moot.** No published modules ⇒ nothing to pin. Revives only if/when published-module distribution graduates from fog.
- **AWS accounts — Organizations now, playground in its own member account.** The one deliberate "more moving parts" call, taken purely on the retrofit-pain argument: moving stateful resources across accounts later (S3 buckets, the CloudFront/ACM/Route 53 tangle) is a genuine migration, whereas enabling Organizations + a dedicated member account is near-free today. **Project-two becomes a new account, not a migration.** `18`/`19`'s single-account SSO assumptions now mean the *playground* account.
- **Cross-project state — state bucket per account.** Completes the account-per-project isolation: each member account hand-bootstraps its own state bucket, so a blast in one project's state can't touch another's and per-account SSO creds can't reach another account's state. Costs one hand-bootstrap per new account — `18`'s already-accepted "bootstrap the state bucket by hand" gesture, repeated per account. **`18`'s per-component state split (S3 `use_lockfile`, never committed) is unchanged *inside* the playground account.**

**Revises `18`/`19`:** their single-account framing is now explicitly the playground's own member account under an Org; the state bucket lives in that account. **Unblocks nothing new.** **Fog:** published-module distribution + its versioning (revives with a real second consuming repo).
