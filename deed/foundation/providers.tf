provider "aws" {
  region = "eu-west-2"

  default_tags {
    tags = {
      Project   = "playground"
      Component = "deed"
      ManagedBy = "terraform"
    }
  }
}

# deed authenticates to the operator's existing single account via ambient SSO
# credentials (AWS_PROFILE). These data sources create nothing; they only report
# where the credentials landed, surfaced through outputs as a wrong-account guard.
data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
