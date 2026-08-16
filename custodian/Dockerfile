# Wraps the static ARM64 release binary that `just build-custodian-release`
# produces (CGO_ENABLED=0 GOOS=linux GOARCH=arm64) into a runnable image for
# the box's Graviton t4g arch. custodian stays agnostic to the deploy shape:
# this image carries only the binary — Litestream is a separate sidecar, and
# compose/nginx/push all live in deed, not here.
#
#   just build-custodian-release
#   docker build --platform linux/arm64 -t custodian custodian
#
# distroless/static gives the minimal base a pure static binary needs: it
# carries CA certs (custodian's S3, source, and OTLP edges all speak TLS) and
# tzdata, runs as an unprivileged user, and ships no shell or package manager.
FROM --platform=linux/arm64 gcr.io/distroless/static-debian12:nonroot

# Every config value and secret is read from the environment at startup via
# os.Getenv (see internal/config); nothing is baked in and nothing is injected
# at runtime by this image. The listen address defaults to :8080.
EXPOSE 8080

# The nonroot user's home is writable by it; run there so custodian's default
# CUSTODIAN_DB_PATH (custodian.db, relative) lands somewhere it can create the
# file. A real deploy overrides the path onto the Litestream-replicated volume.
WORKDIR /home/nonroot

# --chmod pins the execute bit rather than inheriting the host context's mode,
# which on Windows/NTFS builders can arrive without it; the distroless base has
# no shell to repair a non-executable entrypoint at runtime.
COPY --chmod=0755 custodian /usr/local/bin/custodian

ENTRYPOINT ["/usr/local/bin/custodian"]
