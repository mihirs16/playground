# foundation creates no resources — it wires deed to the operator's existing
# account and proves the S3 backend works (a clean, no-change plan). These
# outputs surface which account the ambient SSO credentials resolved to, so an
# apply into the wrong account is caught by eye.

output "account_id" {
  description = "Account ID the ambient credentials resolved to; the sibling components provision into it."
  value       = data.aws_caller_identity.current.account_id
}

output "region" {
  description = "Region deed provisions into."
  value       = data.aws_region.current.name
}
