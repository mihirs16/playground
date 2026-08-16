# 02 — Compute shell: EC2 box, instance profile, Docker bootstrap & SSM bootstrap secrets

**What to build:** The durable box custodian's first version comes up on, with the
startup identity it needs to boot healthily. `deed` provisions an EC2 `t4g.micro`
+ EBS root volume whose `user_data` installs Docker, the compose plugin, and the
AWS CLI and stops at "Docker engine running" — no reaching up into the app layer —
plus the named Docker volume the SQLite file lives in so custodian's data survives
container recreation. The box's only AWS identity is an IAM instance profile;
there is no long-lived AWS key on the box. The same ticket provisions the
**bootstrap secrets** custodian needs at startup — the admin-token hash and the
Grafana Cloud OTLP credential — as SSM `SecureString` parameters under a shared
path, their values supplied via the git-ignored tfvars (landing in encrypted
state, accepted), and grants the instance profile a **path-wildcard read** over
that prefix so adding a bootstrap secret later needs no policy edit. `deed`
delivers *authorization to read*, never the runtime injection of values — the
deploy wrapper does the SSM→env step on the box, not `deed`.

**Blocked by:** 01.

**Status:** done

- [x] EC2 `t4g.micro` + EBS provisioned; `user_data` installs Docker + compose plugin + AWS CLI and stops at "Docker engine running" — `deed/compute/main.tf` (`aws_instance.box`, gp3 `root_block_device`), `deed/compute/user_data.sh`
- [x] Named Docker volume for the SQLite file exists and survives container recreation — `deed/compute/user_data.sh` (`docker volume create`), name in `deed/compute/variables.tf`
- [x] Box's AWS access comes solely from an IAM instance profile; no long-lived AWS key resource exists on the box — `deed/compute/main.tf` (`aws_iam_instance_profile.box`, IMDSv2 `http_tokens = "required"`; no `aws_iam_access_key`)
- [x] SSM `SecureString` params for the admin-token hash and OTLP credential are provisioned from git-ignored tfvars — `deed/compute/main.tf` (`aws_ssm_parameter.bootstrap`), `deed/compute/terraform.tfvars.example`
- [x] Instance profile carries a path-scoped read grant over the bootstrap-secret prefix (no per-secret policy edit) — `deed/compute/main.tf` (`read_bootstrap_secrets`, grants the path node **and** `${var.ssm_bootstrap_prefix}/*` so both `GetParametersByPath` discovery and `GetParameter` by-name reads succeed)
- [x] `deed` provisions only the parameters + read grant — no runtime injection of values — no image pull / secret injection / container start in `user_data.sh`

**Human-verified:** applied to account `136102212434` / `eu-west-2` under the `AdministratorAccess` SSO permission set (`PowerUserAccess` lacks `iam:CreateRole`). Box `custodian` is running; via Session Manager (`AmazonSSMManagedInstanceCore`, no SSH key) confirmed `docker` active, the `custodian-data` volume present, and the compose plugin + AWS CLI installed. Instance-profile `custodian-box` reads its bootstrap secrets end-to-end: `aws ssm get-parameters-by-path /playground/custodian/bootstrap --with-decryption` returns all four names. During verification the read grant was corrected — the original single `/*` ARN authorized by-name reads but not `GetParametersByPath` (which authorizes against the path node); the path node was added.
