# AWS Organizations lives in the management account this config authenticates to.
# A future second project becomes a new member account under the same org, so its
# stateful resources are never a cross-account migration away from this one.

resource "aws_organizations_organization" "this" {
  feature_set = "ALL"
}

resource "aws_organizations_account" "playground" {
  name      = "playground"
  email     = var.playground_account_email
  parent_id = aws_organizations_organization.this.roots[0].id

  # Terraform can create a member account but AWS forbids programmatic deletion of
  # a live one; removing this resource must be a deliberate, documented act, never
  # a side effect of a refactor.
  lifecycle {
    prevent_destroy = true
  }
}
