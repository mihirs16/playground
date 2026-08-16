#!/usr/bin/env bash
# custodian's box, brought up to "Docker engine running" and no further. This
# installs the runtime deed is responsible for — Docker, the compose plugin, the
# AWS CLI — and creates the named volume custodian's SQLite file lives in. It
# does not reach up into the app layer: pulling images, injecting secrets, and
# starting containers belong to the deploy wrapper, not to deed.
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y ca-certificates curl gnupg unzip

# Docker from its official apt repo, so the compose plugin comes with it.
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" >/etc/apt/sources.list.d/docker.list
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# AWS CLI v2 for the deploy wrapper's SSM->env step. ARM64 to match Graviton.
curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o /tmp/awscliv2.zip
unzip -q /tmp/awscliv2.zip -d /tmp
/tmp/aws/install --update
rm -rf /tmp/aws /tmp/awscliv2.zip

systemctl enable --now docker

docker volume create ${docker_volume_name}
