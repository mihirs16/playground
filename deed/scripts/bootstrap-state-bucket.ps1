<#
.SYNOPSIS
  Bootstrap the Terraform state bucket by hand -- the one imperative gesture in
  deed. Terraform never creates, imports, or manages this bucket (a bucket that
  holds its own state is a bootstrap cycle); this script stands it up once.

.DESCRIPTION
  Idempotent: safe to re-run. Run under short-lived SSO credentials for the
  account that owns the state (set $env:AWS_PROFILE first).

.EXAMPLE
  ./bootstrap-state-bucket.ps1

.EXAMPLE
  ./bootstrap-state-bucket.ps1 -Bucket my-bucket -Region eu-west-2
#>

[CmdletBinding()]
param(
  [string]$Bucket = "deed-tfstate-playground-euw2",
  [string]$Region = "eu-west-2"
)

$ErrorActionPreference = "Stop"

Write-Host "Bootstrapping state bucket '$Bucket' in '$Region'..."

$exists = $false
try {
  aws s3api head-bucket --bucket $Bucket 2>$null
  if ($LASTEXITCODE -eq 0) { $exists = $true }
} catch {}

if ($exists) {
  Write-Host "Bucket already exists -- reapplying settings (idempotent)."
} else {
  aws s3api create-bucket `
    --bucket $Bucket `
    --region $Region `
    --create-bucket-configuration LocationConstraint=$Region
  if ($LASTEXITCODE -ne 0) { throw "create-bucket failed" }
  Write-Host "Created bucket."
}

aws s3api put-bucket-versioning `
  --bucket $Bucket `
  --versioning-configuration Status=Enabled
if ($LASTEXITCODE -ne 0) { throw "put-bucket-versioning failed" }
Write-Host "Versioning enabled."

aws s3api put-bucket-encryption `
  --bucket $Bucket `
  --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"aws:kms"}}]}'
if ($LASTEXITCODE -ne 0) { throw "put-bucket-encryption failed" }
Write-Host "Default encryption (aws:kms) enabled."

aws s3api put-public-access-block `
  --bucket $Bucket `
  --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
if ($LASTEXITCODE -ne 0) { throw "put-public-access-block failed" }
Write-Host "Public access blocked."

Write-Host "Done. Bucket '$Bucket' is ready; it must match the 'bucket' in each component backend block."
