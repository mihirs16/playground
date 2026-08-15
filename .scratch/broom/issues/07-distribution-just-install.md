# 07 — Distribution: root `just install`

**What to build:** Getting `broom` onto a machine needs only the repo and the Go
toolchain. A root `just install` recipe builds `broom` from source and installs
it, delegating to the Go toolchain from the thin root justfile. There is no
published artifact in v1 — no Homebrew tap, no GoReleaser release binaries, no
`go install` on a public module path; those are deferred until a second machine or
another user needs broom installed rather than built.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] Root `just install` builds `broom` from source and installs the binary
- [ ] No published-artifact machinery introduced — build-from-source only
