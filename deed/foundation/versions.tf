terraform {
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
  }

  # State lives in the hand-bootstrapped bucket described in deed/README.md. This
  # backend only references that bucket — deed never creates, imports, or manages
  # it. Each component is a sibling directory with its own `key`, so the per-
  # component state split is a naming convention, not shared state.
  #
  # use_lockfile is native S3 locking (no DynamoDB table), requires Terraform 1.10+.
  backend "s3" {
    bucket       = "deed-tfstate-playground-euw1"
    key          = "foundation/terraform.tfstate"
    region       = "eu-west-1"
    encrypt      = true
    use_lockfile = true
  }
}
