output "instance_id" {
  description = "custodian box instance ID."
  value       = aws_instance.box.id
}

output "public_ip" {
  description = "Stable Elastic IP of the box — the CloudFront origin address."
  value       = aws_eip.box.public_ip
}

output "instance_profile_name" {
  description = "The box's sole AWS identity."
  value       = aws_iam_instance_profile.box.name
}

output "ecr_repository_url" {
  description = "Registry URL the custodian image is pushed to and the box pulls from."
  value       = aws_ecr_repository.custodian.repository_url
}

output "media_bucket_name" {
  description = "Media bucket name; the deploy wrapper sets it as CUSTODIAN_MEDIA_BUCKET."
  value       = aws_s3_bucket.media.bucket
}

output "sqlite_backup_bucket_name" {
  description = "SQLite-backup bucket name; Litestream's replication target."
  value       = aws_s3_bucket.sqlite_backup.bucket
}

output "ssm_bootstrap_prefix" {
  description = "Prefix the deploy wrapper reads bootstrap secrets from."
  value       = var.ssm_bootstrap_prefix
}

output "bootstrap_secret_names" {
  description = "Provisioned bootstrap-secret parameter names (names only, never values)."
  value       = sort([for p in aws_ssm_parameter.bootstrap : p.name])
}

output "route53_name_servers" {
  description = "Name servers for the hosted zone. Set these at Squarespace to delegate the domain; certificate validation and the live site depend on the delegation."
  value       = aws_route53_zone.playground.name_servers
}

output "cloudfront_distribution_id" {
  description = "The edge distribution ID; the deploy wrapper invalidates paths against it."
  value       = aws_cloudfront_distribution.edge.id
}

output "cloudfront_domain_name" {
  description = "The distribution's CloudFront domain; the domain's A/AAAA aliases resolve here."
  value       = aws_cloudfront_distribution.edge.domain_name
}

output "custodian_url" {
  description = "Where custodian is reachable over HTTPS once delegation is live."
  value       = "https://${var.custodian_domain_name}"
}
