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

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
