# Root convenience verbs. This is a thin layer over each component's native
# toolchain — never the source of truth for how a component builds.

# Recipe bodies are POSIX sh (&&, VAR=x prefixes, cd). Unix uses the default sh.
# Windows has no sh on PATH, so route through the bash that ships with Git —
# keeping one shell dialect for every recipe instead of forking them per OS.
set windows-shell := ["C:/Program Files/Git/bin/bash.exe", "-cu"]

oapi_codegen := "go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1"

# List available recipes.
default:
    @just --list

# Regenerate all code fanned out from the OpenAPI contract.
gen: gen-custodian

# Fan custodian's OpenAPI spec into server interfaces and a vendored client.
gen-custodian:
    cd custodian && {{oapi_codegen}} -config internal/api/api.cfg.yaml openapi/custodian.yaml
    cd custodian && {{oapi_codegen}} -config internal/apiclient/apiclient.cfg.yaml openapi/custodian.yaml

# CI drift gate: regenerate, then fail if anything changed.
gen-check: gen
    git diff --exit-code

# Build a component (default: custodian).
build component="custodian":
    cd {{component}} && CGO_ENABLED=0 go build ./...

# Cross-compile custodian's static ARM64 release binary.
build-custodian-release:
    cd custodian && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o custodian ./cmd/custodian

# Run a component's tests (default: custodian).
test component="custodian":
    cd {{component}} && go test ./...

# Format and vet custodian.
lint:
    cd custodian && gofmt -l . && go vet ./...
