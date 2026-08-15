output "playground_account_id" {
  description = "Account ID of the playground member account; the components in the sibling directories provision into it."
  value       = aws_organizations_account.playground.id
}

output "organization_id" {
  description = "ID of the AWS Organization the playground account belongs to."
  value       = aws_organizations_organization.this.id
}
