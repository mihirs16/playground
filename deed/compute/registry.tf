# The registry custodian's image lives in and the box pulls from with zero static
# credentials. The box authenticates via its instance profile alone — the reason
# GHCR was rejected was that it would have put a long-lived registry PAT on the box.

resource "aws_ecr_repository" "custodian" {
  name = var.ecr_repository_name
}

# Keep image storage flat and cheap as versions accumulate: retain the last N
# tagged images and expire untagged ones (the layers a re-push leaves behind)
# quickly. Untagged expiry is rule 1 so it runs before the count cap.
resource "aws_ecr_lifecycle_policy" "custodian" {
  repository = aws_ecr_repository.custodian.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Expire untagged images after 1 day"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 1
        }
        action = { type = "expire" }
      },
      {
        rulePriority = 2
        description  = "Keep only the most recent ${var.ecr_image_retention_count} images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = var.ecr_image_retention_count
        }
        action = { type = "expire" }
      },
    ]
  })
}

# The box's pull grant. GetAuthorizationToken is a registry-level action that does
# not support resource scoping, so it stands alone against "*"; the layer/image
# reads that follow are scoped to this one repository.
data "aws_iam_policy_document" "ecr_pull" {
  statement {
    sid       = "GetAuthorizationToken"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }

  statement {
    sid = "PullCustodianImage"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage",
    ]
    resources = [aws_ecr_repository.custodian.arn]
  }
}

resource "aws_iam_role_policy" "ecr_pull" {
  name   = "ecr-pull"
  role   = aws_iam_role.box.id
  policy = data.aws_iam_policy_document.ecr_pull.json
}
