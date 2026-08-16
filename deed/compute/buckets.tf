# The durable homes ADR-0001 says to rent because "if it's gone, it's gone": the
# media bucket custodian serves uploads from and the bucket Litestream replicates
# the SQLite file to. Both carry prevent_destroy so no component destroy can
# silently take unrecoverable data, regardless of how deed's state is later split
# across components — the safety invariant lives on the resource, not the layout.

resource "aws_s3_bucket" "media" {
  bucket = var.media_bucket_name

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket" "sqlite_backup" {
  bucket = var.sqlite_backup_bucket_name

  lifecycle {
    prevent_destroy = true
  }
}

# Neither bucket is reached by the public directly. The media bucket is read only
# by its own dedicated CloudFront CDN (cdn.<domain>) via OAC — custodian is never
# on the media byte path (ADR-0002); the CDN distribution and the bucket policy
# granting it GetObject land in a later deed ticket. The backup bucket is an
# internal Litestream target. Both stay fully private: OAC authorizes through a
# bucket policy scoped to the distribution, not public ACLs, so the access block
# below stands.
resource "aws_s3_bucket_public_access_block" "media" {
  bucket                  = aws_s3_bucket.media.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "sqlite_backup" {
  bucket                  = aws_s3_bucket.sqlite_backup.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# The box's read/write grant over both data buckets: custodian's media
# reserve/confirm flow needs object read/write/delete plus a bucket listing, and
# Litestream needs the same shape against its backup target. Object actions are
# scoped to the "/*" arm; ListBucket authorizes against the bucket ARN itself.
data "aws_iam_policy_document" "data_buckets" {
  statement {
    sid     = "ListDataBuckets"
    actions = ["s3:ListBucket"]
    resources = [
      aws_s3_bucket.media.arn,
      aws_s3_bucket.sqlite_backup.arn,
    ]
  }

  statement {
    sid = "ReadWriteDataObjects"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = [
      "${aws_s3_bucket.media.arn}/*",
      "${aws_s3_bucket.sqlite_backup.arn}/*",
    ]
  }
}

resource "aws_iam_role_policy" "data_buckets" {
  name   = "data-buckets-read-write"
  role   = aws_iam_role.box.id
  policy = data.aws_iam_policy_document.data_buckets.json
}
