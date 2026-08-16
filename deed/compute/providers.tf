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

# CloudFront reads its viewer certificate only from us-east-1, wherever the origin
# lives (ADR-0001's cross-region gotcha). This aliased provider exists solely so
# the ACM certificate can be issued there; everything else stays in eu-west-2.
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"

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
