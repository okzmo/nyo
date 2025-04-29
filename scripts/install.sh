#!/usr/bin/env bash
set -euo pipefail

echo "What will be this node's name? (no spaces, dash and underscore allowed)"
read NODE_NAME

if [[ ! "$NODE_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]; then
  echo "Invalid node name."
  exit 1
fi

echo "What's your name? (no spaces, for logs)"
read OWNER_NAME

if [[ ! "$OWNER_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]; then
  echo "Invalid owner name."
  exit 1
fi

echo "What is the ssh key that'll be set as the owner of this node? (e.g. ssh-ed25519 ...)"
read OWNER_SSH_KEY

if [[ ! "$OWNER_SSH_KEY" =~ ^ssh-(rsa|ed25519|ecdsa).* ]]; then
  echo "Invalid SSH key format."
  exit 1
fi

echo "What password for this owner node? (Used mostly for maintenance, will not echo)"
read -s PASSWORD

# Create user and set password
useradd -m -s /bin/bash "$NODE_NAME"
echo "$NODE_NAME:$PASSWORD" | chpasswd

# Setup SSH
USER_HOME="/home/$NODE_NAME"
mkdir -p $USER_HOME/.ssh
echo "$OWNER_SSH_KEY" >"$USER_HOME/.ssh/authorized_keys"
chmod 700 "$USER_HOME/.ssh"
chmod 600 "$USER_HOME/.ssh/authorized_keys"
chown -R "$NODE_NAME:$NODE_NAME" "$USER_HOME/.ssh"

if [ -f /etc/ssh/sshd_config ]; then
  cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak
fi
curl -fsSL "https://raw.githubusercontent.com/okzmo/nyo/refs/heads/master/scripts/stubs/sshd_config" -o /etc/ssh/sshd_config
systemctl restart ssh

apt-get update

# Install crowdsec
curl -s https://install.crowdsec.net | bash

# Install nginx and certbot
apt-get install -y nginx certbot python3-certbot-nginx
cat >/etc/nginx/sites-available/default <<EOF
server {
    listen 80 default_server;
    listen [::]:80 default_server;

    server_name _;

    location / {
        return 200 'Nginx is up!';
        add_header Content-Type text/plain;
    }
}
EOF

nginx -t && systemctl restart nginx

# Install postgresql
apt-get install -y postgresql

# Install redis
apt-get install -y lsb-release curl gpg
curl -fsSL https://packages.redis.io/gpg | gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
chmod 644 /usr/share/keyrings/redis-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/redis.list
apt-get install -y redis-server
systemctl enable redis-server
systemctl start redis-server

# Install asdf
ASDF_VERSION=$(curl -s https://api.github.com/repos/asdf-vm/asdf/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -LO "https://github.com/asdf-vm/asdf/releases/download/${ASDF_VERSION}/asdf-${ASDF_VERSION}-linux-amd64.tar.gz"
tar -xzf asdf-*-linux-amd64.tar.gz
mv asdf /usr/local/bin/
rm asdf-*-linux-amd64.tar.gz

# Setup .nyo directory
mkdir -p $USER_HOME/.nyo
chmod 700 "$USER_HOME/.nyo"
chown -R "$NODE_NAME:$NODE_NAME" "$USER_HOME/.nyo"

# Setup nyo_users file
touch /etc/nyo_users
chown root:root /etc/nyo_users
chmod 640 /etc/nyo_users
echo "$OWNER_NAME $OWNER_SSH_KEY OWNER" >>/etc/nyo_users

# End
echo "Nyo install complete!"
echo "User $NODE_NAME created with SSH access for $OWNER_NAME."
echo "Root SSH login is disabled. You can now connect this node using the nyo CLI."
echo "nyo connect $NODE_NAME@ip"
