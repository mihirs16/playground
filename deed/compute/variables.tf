variable "instance_type" {
  description = "custodian's box. ARM64 Graviton per ADR-0001; resizing is a stop/change/start."
  type        = string
  default     = "t4g.micro"
}

variable "root_volume_size" {
  description = "gp3 root EBS volume size in GiB. Holds the OS and the Docker named volume."
  type        = number
  default     = 20
}

variable "docker_volume_name" {
  description = "Named Docker volume custodian's SQLite file lives in; survives container recreation."
  type        = string
  default     = "custodian-data"
}

variable "ssm_bootstrap_prefix" {
  description = "SSM path prefix the bootstrap secrets live under. The instance profile is granted a wildcard read over this prefix, so a new secret is a tfvars edit, not a policy edit."
  type        = string
  default     = "/playground/custodian/bootstrap"
}

variable "bootstrap_secrets" {
  description = "Bootstrap-secret values custodian reads at startup (admin-token hash, Grafana Cloud OTLP credential). Keys become the leaf of the SSM parameter name under ssm_bootstrap_prefix. Supplied via git-ignored tfvars; lands in encrypted state, accepted."
  type        = map(string)
  sensitive   = true
}
