#!/bin/bash
# Copyright (c) 2025 ne43, Inc.
# Licensed under the MIT License. See LICENSE in the project root for details.


user=$(whoami)
pubkey=$(cat ~/.ssh/id_ed25519.pub)


cat <<EOF
#!/bin/bash

echo "Starting FOKS Cloud Init Script ----"

apt update
useradd -m -s /bin/bash foks
useradd -m -s /bin/bash ${user}

# Docker
apt-get update
apt-get -y install ca-certificates curl
apt-get -y install locales-all xxd postgresql-client
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo \
  "deb [arch=\$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
  \$(. /etc/os-release && echo "\$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update
apt-get -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
groupadd docker
usermod -aG docker foks
newgrp docker

echo "${user} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/91-local-admin-${user}

sudo -u foks bash -s << 'EOF2'
cd \${HOME}
mkdir .ssh
chmod 755 .ssh
mkdir -p run/bin run/scripts
AK=.ssh/authorized_keys
echo "${pubkey}" > \${AK}
chown foks:foks \${AK}
chmod 640 \${AK}
EOF2

sudo -u ${user} bash -s << 'EOF3'
cd \${HOME}
mkdir .ssh
chmod 755 .ssh
AK=.ssh/authorized_keys
echo "${pubkey}" > \${AK}
chown ${user}:${user} \${AK}
chmod 640 \${AK}
EOF3

EOF

