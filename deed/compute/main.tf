# custodian's compute shell: the box, its boot identity, and the bootstrap
# secrets it reads at startup. deed delivers authorization to read those secrets,
# never the runtime injection of their values.

data "aws_vpc" "default" {
  default = true
}

# Canonical's Ubuntu 24.04 LTS for ARM64 — the box is Graviton (t4g).
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# Egress-only: the box reaches SSM, ECR, and Docker's apt repo outbound; nothing
# reaches in yet. Public ingress arrives with the edge (ticket 05).
resource "aws_security_group" "box" {
  name        = "custodian-box"
  description = "custodian box: all egress, no ingress until the edge is wired"
  vpc_id      = data.aws_vpc.default.id

  egress {
    description = "All outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "custodian-box"
  }
}

data "aws_iam_policy_document" "box_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "box" {
  name               = "custodian-box"
  assume_role_policy = data.aws_iam_policy_document.box_assume.json
}

# A path-wildcard read over the bootstrap prefix: adding a bootstrap secret later
# is a tfvars edit, never a policy edit. Reading a SecureString also needs
# kms:Decrypt; the account's default aws/ssm key grants that when the call comes
# through SSM, so no explicit KMS statement is required here.
#
# Two resource ARNs, not one: GetParameter/GetParameters authorize against each
# child parameter (the "/*" arm), but GetParametersByPath authorizes against the
# path node itself (no trailing slash), which "/*" does not match. Both are needed
# so the wrapper can discover secrets by path, not just read them by exact name.
data "aws_iam_policy_document" "read_bootstrap_secrets" {
  statement {
    sid = "ReadBootstrapSecrets"
    actions = [
      "ssm:GetParameter",
      "ssm:GetParameters",
      "ssm:GetParametersByPath",
    ]
    resources = [
      "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter${var.ssm_bootstrap_prefix}",
      "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter${var.ssm_bootstrap_prefix}/*",
    ]
  }
}

resource "aws_iam_role_policy" "read_bootstrap_secrets" {
  name   = "read-bootstrap-secrets"
  role   = aws_iam_role.box.id
  policy = data.aws_iam_policy_document.read_bootstrap_secrets.json
}

# Keyless access to the box: Session Manager, not SSH. This lets the SSM agent
# register and the operator reach the box to run the deploy wrapper, keeping the
# "no long-lived key on the box" story intact.
resource "aws_iam_role_policy_attachment" "ssm_core" {
  role       = aws_iam_role.box.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "box" {
  name = "custodian-box"
  role = aws_iam_role.box.name
}

# Bootstrap secrets, one SecureString per map entry under the shared prefix.
# Values come from git-ignored tfvars and land in encrypted state (accepted).
resource "aws_ssm_parameter" "bootstrap" {
  # Only the values are secret; the keys are parameter names. nonsensitive() on
  # the key set keeps them usable as instance keys without leaking any value.
  for_each = nonsensitive(toset(keys(var.bootstrap_secrets)))

  name  = "${var.ssm_bootstrap_prefix}/${each.key}"
  type  = "SecureString"
  value = var.bootstrap_secrets[each.key]
}

resource "aws_instance" "box" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  iam_instance_profile   = aws_iam_instance_profile.box.name
  vpc_security_group_ids = [aws_security_group.box.id]

  user_data = templatefile("${path.module}/user_data.sh", {
    docker_volume_name = var.docker_volume_name
  })

  root_block_device {
    volume_type = "gp3"
    volume_size = var.root_volume_size
    encrypted   = true
  }

  # IMDSv2 only — the instance profile is the box's sole AWS identity, reached
  # through the metadata credential chain. hop_limit 2 so custodian, running in a
  # container, can still reach the endpoint (one extra hop crossing the container
  # network).
  metadata_options {
    http_tokens                 = "required"
    http_endpoint               = "enabled"
    http_put_response_hop_limit = 2
  }

  tags = {
    Name = "custodian"
  }
}
