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

**Status:** ready-for-agent

- [ ] EC2 `t4g.micro` + EBS provisioned; `user_data` installs Docker + compose plugin + AWS CLI and stops at "Docker engine running"
- [ ] Named Docker volume for the SQLite file exists and survives container recreation
- [ ] Box's AWS access comes solely from an IAM instance profile; no long-lived AWS key resource exists on the box
- [ ] SSM `SecureString` params for the admin-token hash and OTLP credential are provisioned from git-ignored tfvars
- [ ] Instance profile carries a path-scoped read grant over the bootstrap-secret prefix (no per-secret policy edit)
- [ ] `deed` provisions only the parameters + read grant — no runtime injection of values
