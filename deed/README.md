# deed

The declared record of the ground the playground sits on. Flat in-repo HCL (no
modules), one component per directory, each its own Terraform root wired to a
shared S3 backend.

## Layout

```
deed/
  foundation/   AWS Organizations + the playground member account
```

Every component is a sibling directory with its own backend `key`
(`<component>/terraform.tfstate`). This is the per-component state split: a
`destroy` in one component can never touch another's state. New components
follow the same shape — a flat root, its own `key`, no shared module.

## The one imperative gesture: bootstrap the state bucket by hand

The backend references an S3 bucket that Terraform **never creates, imports, or
manages** — a bucket managing the state that describes it is a bootstrap cycle.
Create it by hand once, then only reference it. Run these against the account
that owns the state (short-lived SSO credentials, see below):

```sh
BUCKET=deed-tfstate-playground-euw1
REGION=eu-west-1

aws s3api create-bucket \
  --bucket "$BUCKET" \
  --region "$REGION" \
  --create-bucket-configuration LocationConstraint="$REGION"

aws s3api put-bucket-versioning \
  --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled

aws s3api put-bucket-encryption \
  --bucket "$BUCKET" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"aws:kms"}}]}'

aws s3api put-public-access-block \
  --bucket "$BUCKET" \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
```

The bucket name here must match the `bucket` in each component's `backend "s3"`
block. State locking is native (`use_lockfile = true`) — no DynamoDB table.

## Credentials: SSO only

Apply is laptop-only under short-lived IAM Identity Center (SSO) credentials.
There is no static IAM access key on disk anywhere. Sign in before running any
recipe that touches AWS:

```sh
aws sso login --profile <your-sso-profile>
export AWS_PROFILE=<your-sso-profile>
```

CI never applies. It runs `fmt` / `validate` / `plan` only (see
`.github/workflows/deed.yml`), so a drifted or broken config fails the plan gate
before it can merge.

## Recipes

From the repo root:

```sh
just deed-fmt         # gofmt-equivalent: canonical HCL formatting
just deed-validate    # config is internally consistent (no backend/creds needed)
just deed-plan        # plan against the real backend (needs SSO credentials)
just deed-apply       # apply a component (needs SSO credentials); CI never runs this
```

`deed-plan` and `deed-apply` take a component name, defaulting to `foundation`:

```sh
just deed-plan foundation
just deed-apply foundation
```
