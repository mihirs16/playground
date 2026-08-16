output "instance_id" {
  description = "custodian box instance ID."
  value       = aws_instance.box.id
}

output "public_ip" {
  description = "Ephemeral public IP of the box. Stable addressing arrives with the edge (ticket 05)."
  value       = aws_instance.box.public_ip
}

output "instance_profile_name" {
  description = "The box's sole AWS identity."
  value       = aws_iam_instance_profile.box.name
}

output "ssm_bootstrap_prefix" {
  description = "Prefix the deploy wrapper reads bootstrap secrets from."
  value       = var.ssm_bootstrap_prefix
}

output "bootstrap_secret_names" {
  description = "Provisioned bootstrap-secret parameter names (names only, never values)."
  value       = sort([for p in aws_ssm_parameter.bootstrap : p.name])
}
