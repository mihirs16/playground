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

variable "ecr_repository_name" {
  description = "ECR repository custodian's image is pushed to and the box pulls from via its instance profile."
  type        = string
  default     = "custodian"
}

variable "ecr_image_retention_count" {
  description = "How many recent images the lifecycle policy keeps. Older images (and untagged ones sooner) expire, keeping storage flat as versions accumulate."
  type        = number
  default     = 10
}

variable "media_bucket_name" {
  description = "S3 bucket custodian serves media from and broom writes uploads to. Globally unique; carries prevent_destroy since lost media is unrecoverable."
  type        = string
  default     = "custodian-media-playground-euw2"
}

variable "sqlite_backup_bucket_name" {
  description = "S3 bucket Litestream replicates the SQLite file to. Globally unique; carries prevent_destroy since it is the only off-box copy of custodian's records."
  type        = string
  default     = "custodian-sqlite-backup-playground-euw2"
}

variable "zone_name" {
  description = "The apex of the Route 53 hosted zone deed creates. Registration stays at Squarespace; the zone's name servers (route53_name_servers output) must be delegated there before an apply can complete certificate validation. The apex itself is persona's (the front-facing website); custodian is a subdomain under it."
  type        = string
  default     = "mihirsingh.dev"
}

variable "custodian_domain_name" {
  description = "The domain custodian is served at through the edge — the distribution alias and the ACM certificate subject. A subdomain of zone_name (its records live in that hosted zone)."
  type        = string
  default     = "custodian.mihirsingh.dev"
}

variable "cdn_domain_name" {
  description = "The dedicated media CDN hostname — the alias and ACM subject of the second CloudFront distribution that fronts the private media bucket via OAC (ADR-0002). A subdomain of zone_name; custodian records absolute urls under https://<this>."
  type        = string
  default     = "cdn.mihirsingh.dev"
}

variable "bootstrap_secrets" {
  description = "Startup secrets custodian reads from the environment. Each key is the exact env var name custodian expects (custodian/internal/config/config.go) and becomes the SSM parameter leaf under ssm_bootstrap_prefix, so the deploy wrapper's SSM->env step exports each leaf verbatim with no mapping table. Supplied via git-ignored tfvars; lands in encrypted state, accepted."
  type        = map(string)
  sensitive   = true
}
