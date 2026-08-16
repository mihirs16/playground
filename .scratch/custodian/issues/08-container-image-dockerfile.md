# 08 — Container image: Dockerfile for the static ARM64 binary

**What to build:** custodian gets the one packaging artifact that legitimately
lives with its code — a `Dockerfile` that turns the static ARM64 release binary
into a runnable `linux/arm64` image. custodian stays agnostic to the deploy shape
(`custodian.md:157-160`): the Dockerfile only wraps the binary the existing
`just build-custodian-release` recipe already produces (`CGO_ENABLED=0
GOOS=linux GOARCH=arm64`), on a minimal base, exposing nothing custodian doesn't
already read from `os.Getenv`. No compose, no nginx, no push — those are deploy
concerns (`deed/07`), deliberately not custodian's.

The image is a **pure static binary** — Litestream runs as a separate sidecar
container per `19` (`deed.md:255-257`), so it does not belong in this image.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] `Dockerfile` builds a `linux/arm64` image from the static release binary (matches the box's Graviton `t4g` arch)
- [ ] Image contains only custodian's binary on a minimal base — no Litestream, no compose/nginx artifacts
- [ ] Container reads all config/secrets from the environment (`os.Getenv`); the image bakes in no secret and no runtime injection
- [ ] Image runs custodian and the binary starts (a local `docker run` smoke, env supplied by hand)
