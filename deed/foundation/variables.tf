variable "playground_account_email" {
  description = "Root email for the playground member account. Must be unique across all AWS accounts; supplied via the git-ignored terraform.tfvars."
  type        = string
}
