# Monorepo tooling and layout

Type: grilling
Status: open
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
