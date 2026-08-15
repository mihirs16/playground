provider "aws" {
  region = "eu-west-1"

  default_tags {
    tags = {
      Project   = "playground"
      Component = "deed"
      ManagedBy = "terraform"
    }
  }
}
