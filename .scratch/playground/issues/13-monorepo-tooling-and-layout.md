# Monorepo tooling and layout

Type: grilling
Status: resolved
Blocked by: 07

## Question

How is the polyglot monorepo organised and orchestrated?

- **Directory layout** — what's the top level? `apps/` and `packages/`, or flat directories named for the components (`custodian/`, `broom/`, `blank/`, `persona/`, `deed/`)? Flat and named is more legible for five known components; conventional groupings pay off at larger scale.
- **Where `deed` sits** — `01` established it as the fifth component and assumed the monorepo, but a Terraform root with its own state and destroy-everything blast radius is the one component with a real argument for living apart. If it stays in, does anything stop a careless `apply` from a component-scoped CI job?
- **Task orchestration** — does anything coordinate builds across languages? Options range from nothing at all (each component has its own toolchain and you `cd` into it), through a `Makefile` or `just` at the root, to a real polyglot build system (Bazel, Pants, Nx with custom executors). The honest answer for two languages and four components may well be "a Makefile".
- **Node workspaces** — `blank` and probably persona are npm packages. pnpm workspaces, npm workspaces, or independent installs? The deprecated repo used pnpm, so there's precedent.
- **Shared artifacts across the language boundary** — where does a generated API client live, and who generates it? This is the one place the polyglot split actually costs something concrete.
- **Repo hygiene** — one `.gitignore` or per-component, formatter and linter config per language, whether a single pre-commit hook can cover everything.
- **Does the deprecated repo's history come along**, or is this a clean start with the old repo left archived?

## Context

Blocked on `07`, because the language set determines whether this is a genuine polyglot orchestration problem or just "npm workspaces plus one Go module".

`01` added a fifth component, `deed` (Terraform/HCL), and a third language to place. It also made deployment concrete — an EC2 box, an S3 bucket, a CloudFront distribution — which means whatever this ticket decides about orchestration now has a real deploy target to serve, and `19` is waiting on it.

Since polyglot turned out to be incidental rather than a goal, resist over-engineering here. Bazel for a two-language personal monorepo is a trap dressed as rigour. The bar for adopting a build system should be a concrete pain it removes, not tidiness.

Note the repo this map lives in is currently a bare `git init` on `master` with no commits, containing only `CLAUDE.md`, `docs/agents/`, and `.scratch/`. Whether `master` is renamed to `main` is a trivial call to fold in here.

## Answer

**Flat, named top level.** `custodian/ broom/ blank/ persona/ deed/` at the root — the filesystem mirrors the project's ubiquitous language. No `apps/`/`packages/` grouping; five known named components read better flat than sorted into conventional buckets that only pay off at larger scale.

**`deed` lives in-repo, but nothing applies it today.** It's atomically versioned with the code it provisions. CI for `deed` caps at `terraform fmt -check` / `validate` / `plan` (plan may post output, never applies). A *gated* CI-apply path is an explicit future possibility, not foreclosed — its mechanics are deferred to `18` (deed state backend and apply authority). Blast radius is contained by apply-authority, not by physical repo separation.

**Task orchestration: a thin root `justfile`.** A convenience layer of memorable verbs (`just build custodian`, `just test`, `just gen`) that delegates to each component's native toolchain; never the source of truth for how a component builds. No Bazel/Nx/Pants — for two-to-three languages and five components there's no dependency-graph or cache-invalidation pain to justify one. The bar to graduate is a concrete measured pain (e.g. CI cross-component cache misses), which is fog until CI exists. `just` over `make` to avoid make's rebuild semantics and whitespace footguns.

**Node side: pnpm workspaces, TS components only.** Precedent from the deprecated repo; pnpm's strict non-flat `node_modules` catches phantom dependencies, which matters because `blank` publishes publicly and must not lean on a persona-provided dep. `pnpm-workspace.yaml` lists the TS components (`blank`, persona, and `broom` iff `17` picks TS). Go (`custodian`) and HCL (`deed`) have no `package.json` and aren't listed, so pnpm is completely blind to them — no hoisting, no lockfile entanglement. The two toolchains coexist at the same flat level without interacting.

**Generated API client: one spec, per-consumer vendored clients, no shared package.** The single OpenAPI 3.1 `.yaml` (from `09`) is the source of truth and lives under custodian (`custodian/openapi/…`) — the server owns the contract. `just gen` fans out a generated client **into each consumer that needs one** — `broom`, persona, and custodian itself for tests — each vendored into that component's own tree in its own language. Deliberately **no standalone shared client package**: the coupling is to the spec file, not to a cross-component artifact. Generated code is **committed** (fresh clone builds with zero codegen; contract changes are reviewable in PR diffs); **CI enforces zero drift** (`just gen` then `git diff --exit-code`).

**Repo hygiene: per-language, coordinated by convention, no pre-commit hook in v1.** A small root `.gitignore` for universal cruft (`.env`, `node_modules`, `dist`, OS/editor files) plus per-component ignores where a toolchain has specific artifacts — Terraform state (`*.tfstate`, `.terraform/`) **never committed**. Formatting/linting is per-language and idiomatic (`gofmt` + `golangci-lint`, Prettier + ESLint, `terraform fmt`), exposed as `just` recipes and run in CI. No pre-commit hook for now (nothing blocks a local commit); the `pre-commit` framework is explicitly rejected as another toolchain to install. A thin hook can be added later if drift becomes a problem.

**Clean git start, branch renamed to `main`.** The deprecated repo (`mihirs16/mihirsingh.dev-deprecated`) is left archived and linked from the README for provenance — its history is *not* imported (its value, design tokens + content model, is already harvested into `CONTEXT.md`/ADRs; the map already rules out incrementally migrating it). `master` is renamed to `main` now, while the repo has zero commits, matching the map's PR convention.

**Unblocks `19`** (deed provisioning boundary), which now has the layout and orchestration target it was waiting on.
