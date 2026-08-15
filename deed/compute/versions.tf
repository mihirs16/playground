terraform {
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
  }

  # Sibling of foundation: same bucket, its own key, so a destroy here can never
  # touch another component's state. use_lockfile is native S3 locking (no
  # DynamoDB table), requires Terraform 1.10+.
  backend "s3" {
    bucket       = "deed-tfstate-playground-euw2"
    key          = "compute/terraform.tfstate"
    region       = "eu-west-2"
    encrypt      = true
    use_lockfile = true
  }
}
