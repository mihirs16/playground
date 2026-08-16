#!/usr/bin/env bash
# The deploy wrapper: the app-layer step deed deliberately does not do
# (deed.md:273-281, story 31). It runs ON THE BOX, invoked by `just
# deploy-custodian` over SSM Run Command — never over SSH. deed grants the box
# authorization to read the bootstrap secrets; turning that authorization into
# running containers is this script's job, not Terraform's.
#
# It expects the deploy artifacts (compose.yml, nginx.conf, litestream.yml) to
# sit alongside it, and these variables in the environment, exported by the SSM
# command from deed's Terraform outputs:
#
#   CUSTODIAN_IMAGE                 ECR image ref incl. tag, e.g. <acct>.dkr.ecr.<region>.amazonaws.com/custodian:<tag>
#   CUSTODIAN_SSM_BOOTSTRAP_PREFIX  SSM path the bootstrap secrets live under
#   CUSTODIAN_MEDIA_BUCKET          non-secret config compose interpolates
#   CUSTODIAN_MEDIA_CDN_BASE
#   CUSTODIAN_CORS_ALLOWLIST
#   CUSTODIAN_SQLITE_BACKUP_BUCKET  Litestream's replication target
#   AWS_REGION
set -euo pipefail

require() {
    if [ -z "${!1:-}" ]; then
        echo "deploy-wrapper: missing required env $1" >&2
        exit 64
    fi
}
require CUSTODIAN_IMAGE
require CUSTODIAN_SSM_BOOTSTRAP_PREFIX
require AWS_REGION

cd "$(dirname "$0")"

# SSM -> env onto tmpfs. /run is RAM-only, so the plaintext secret never touches
# a disk-backed filesystem. Each parameter leaf under the prefix IS the env var
# name custodian expects (deed's bootstrap_secrets keys are exactly those names),
# so the export is verbatim with no mapping table.
env_file=/run/custodian.env
umask 077
tmp_env=$(mktemp /run/custodian.env.XXXXXX)
trap 'rm -f "$tmp_env"' EXIT

names=$(aws ssm get-parameters-by-path \
    --path "$CUSTODIAN_SSM_BOOTSTRAP_PREFIX" \
    --with-decryption --recursive \
    --query 'Parameters[].Name' --output text)

for name in $names; do
    leaf=${name##*/}
    value=$(aws ssm get-parameter --name "$name" --with-decryption \
        --query 'Parameter.Value' --output text)
    printf '%s=%s\n' "$leaf" "$value" >>"$tmp_env"
done

mv "$tmp_env" "$env_file"
trap - EXIT

# Pull under the instance profile: deed grants the box GetAuthorizationToken +
# repo-scoped image reads, so this login carries no static registry credential.
registry=${CUSTODIAN_IMAGE%%/*}
aws ecr get-login-password --region "$AWS_REGION" \
    | docker login --username AWS --password-stdin "$registry"

# The rollout itself. compose reads the secrets from env_file and the non-secret
# ${...} values from this process's environment.
docker compose pull
docker compose up -d --remove-orphans
