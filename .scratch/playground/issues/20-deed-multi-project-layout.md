# deed: multi-project layout

Type: grilling
Status: open
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
