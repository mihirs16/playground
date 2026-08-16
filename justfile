# Root convenience verbs. This is a thin layer over each component's native
# toolchain — never the source of truth for how a component builds.

# Recipe bodies are POSIX sh (&&, VAR=x prefixes, cd). Unix uses the default sh.
# Windows has no sh on PATH, so route through the bash that ships with Git —
# keeping one shell dialect for every recipe instead of forking them per OS.
[windows]
set shell := ["C:/Program Files/Git/bin/bash.exe", "-cu"]

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

# Build custodian's linux/arm64 image and push it to ECR under SSO — no Terraform.
# Registry URL is deed/compute's ecr_repository_url output; region read from its host.
push-custodian tag="latest": build-custodian-release
    set -eu; \
    repo="$(cd deed/compute && terraform output -raw ecr_repository_url)"; \
    region="$(printf '%s' "$repo" | sed -E 's/.*\.dkr\.ecr\.([^.]+)\..*/\1/')"; \
    docker build --platform linux/arm64 -t "$repo:{{tag}}" custodian; \
    aws ecr get-login-password --region "$region" \
        | docker login --username AWS --password-stdin "${repo%%/*}"; \
    docker push "$repo:{{tag}}"

# Roll custodian out on the box via SSM Run Command — no SSH, no Terraform.
# Ships the deploy artifacts + wrapper, then the wrapper does SSM->tmpfs + compose up.
deploy-custodian tag="latest":
    set -eu; \
    repo="$(cd deed/compute && terraform output -raw ecr_repository_url)"; \
    instance="$(cd deed/compute && terraform output -raw instance_id)"; \
    media_bucket="$(cd deed/compute && terraform output -raw media_bucket_name)"; \
    backup_bucket="$(cd deed/compute && terraform output -raw sqlite_backup_bucket_name)"; \
    ssm_prefix="$(cd deed/compute && terraform output -raw ssm_bootstrap_prefix)"; \
    region="$(printf '%s' "$repo" | sed -E 's/.*\.dkr\.ecr\.([^.]+)\..*/\1/')"; \
    image="$repo:{{tag}}"; \
    payload="$(tar -czf - -C deploy compose.yml nginx.conf litestream.yml deploy-wrapper.sh | base64 | tr -d '\n')"; \
    remote="$(printf '%s\n' \
        "set -eu" \
        "install -d -m 0755 /opt/custodian" \
        "printf '%s' '$payload' | base64 -d | tar -xzf - -C /opt/custodian" \
        "export CUSTODIAN_IMAGE='$image'" \
        "export CUSTODIAN_SSM_BOOTSTRAP_PREFIX='$ssm_prefix'" \
        "export CUSTODIAN_MEDIA_BUCKET='$media_bucket'" \
        "export CUSTODIAN_SQLITE_BACKUP_BUCKET='$backup_bucket'" \
        "export AWS_REGION='$region'" \
        "bash /opt/custodian/deploy-wrapper.sh")"; \
    remote_b64="$(printf '%s' "$remote" | base64 | tr -d '\n')"; \
    params="$(printf '{"commands":["echo %s | base64 -d | bash"]}' "$remote_b64")"; \
    command_id="$(aws ssm send-command \
        --region "$region" \
        --instance-ids "$instance" \
        --document-name AWS-RunShellScript \
        --comment "custodian rollout {{tag}}" \
        --parameters "$params" \
        --query 'Command.CommandId' --output text)"; \
    echo "SSM command $command_id sent to $instance ({{tag}}); waiting for it to finish..."; \
    aws ssm wait command-executed --region "$region" --command-id "$command_id" --instance-id "$instance" || true; \
    aws ssm get-command-invocation --region "$region" --command-id "$command_id" --instance-id "$instance" \
        --query '{Status:Status,Stdout:StandardOutputContent,Stderr:StandardErrorContent}' --output text; \
    status="$(aws ssm get-command-invocation --region "$region" --command-id "$command_id" --instance-id "$instance" --query 'Status' --output text)"; \
    [ "$status" = "Success" ] || { echo "rollout failed on $instance: $status" >&2; exit 1; }

# Canonically format all deed HCL in place.
deed-fmt:
    cd deed && terraform fmt -recursive

# Check that all deed HCL is canonically formatted (CI gate; no writes).
deed-fmt-check:
    cd deed && terraform fmt -check -recursive

# Validate a deed component's config; no backend or credentials needed (default: foundation).
deed-validate component="foundation":
    cd deed/{{component}} && terraform init -backend=false -input=false && terraform validate

# Plan a deed component against the real S3 backend (needs SSO credentials).
deed-plan component="foundation":
    cd deed/{{component}} && terraform init -input=false && terraform plan -input=false

# Apply a deed component (needs SSO credentials). CI never runs this.
deed-apply component="foundation":
    cd deed/{{component}} && terraform init -input=false && terraform apply -input=false
