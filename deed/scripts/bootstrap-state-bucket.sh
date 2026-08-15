#!/usr/bin/env bash
# Bootstrap the Terraform state bucket by hand — the one imperative gesture in
# deed. Terraform never creates, imports, or manages this bucket (a bucket that
# holds its own state is a bootstrap cycle); this script stands it up once.
#
# Idempotent: safe to re-run. Run under short-lived SSO credentials for the
# account that owns the state (export AWS_PROFILE first).
#
# Usage:
#   ./bootstrap-state-bucket.sh
#   BUCKET=my-bucket REGION=eu-west-2 ./bootstrap-state-bucket.sh

set -euo pipefail

BUCKET="${BUCKET:-deed-tfstate-playground-euw2}"
REGION="${REGION:-eu-west-2}"

echo "Bootstrapping state bucket '$BUCKET' in '$REGION'..."

if aws s3api head-bucket --bucket "$BUCKET" 2>/dev/null; then
  echo "Bucket already exists — reapplying settings (idempotent)."
else
  aws s3api create-bucket \
    --bucket "$BUCKET" \
    --region "$REGION" \
    --create-bucket-configuration LocationConstraint="$REGION"
  echo "Created bucket."
fi

aws s3api put-bucket-versioning \
  --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled
echo "Versioning enabled."

aws s3api put-bucket-encryption \
  --bucket "$BUCKET" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"aws:kms"}}]}'
echo "Default encryption (aws:kms) enabled."

aws s3api put-public-access-block \
  --bucket "$BUCKET" \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
echo "Public access blocked."

echo "Done. Bucket '$BUCKET' is ready; it must match the 'bucket' in each component's backend block."
