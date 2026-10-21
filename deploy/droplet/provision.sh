#!/usr/bin/env bash
# One-time provisioning for the Hearth deploy on an existing droplet.
# Idempotent — safe to re-run.
#
# Prerequisites on the droplet:
#   - Caddy already installed and running (handles TLS for the other subdomains)
#   - Caddyfile set up to `import /etc/caddy/sites/*` (true for this droplet)
#
# Usage, as root on the droplet:
#   git clone https://github.com/dopatraman/hearth /tmp/hearth
#   cd /tmp/hearth
#   sudo bash deploy/droplet/provision.sh

set -euo pipefail

if [[ $EUID -ne 0 ]]; then
    echo "run as root (try: sudo bash $0)" >&2
    exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

for f in deploy/hearth.service deploy/Caddyfile.hearth deploy/sudoers.d-hearth; do
    if [[ ! -f "$REPO_ROOT/$f" ]]; then
        echo "missing expected file: $REPO_ROOT/$f" >&2
        exit 1
    fi
done

echo "==> create deploy user"
if ! id deploy &>/dev/null; then
    useradd --create-home --shell /bin/bash deploy
    echo "    created"
else
    echo "    already exists"
fi

echo "==> install hearth-deploy ssh key for the deploy user"
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
HEARTH_DEPLOY_KEY='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPqM4pLk5OyM+GxbDgUP3yUMQh8auy1ayEaJHjI/hJNR hearth-deploy@github-actions'
AUTH_KEYS="/home/deploy/.ssh/authorized_keys"
touch "$AUTH_KEYS"
if ! grep -qF "$HEARTH_DEPLOY_KEY" "$AUTH_KEYS"; then
    echo "$HEARTH_DEPLOY_KEY" >> "$AUTH_KEYS"
    echo "    key appended"
else
    echo "    key already present"
fi
chmod 600 "$AUTH_KEYS"
chown deploy:deploy "$AUTH_KEYS"

echo "==> create /opt/hearth tree"
install -d -m 755 -o deploy -g deploy /opt/hearth
install -d -m 755 -o deploy -g deploy /opt/hearth/bin
install -d -m 755 -o deploy -g deploy /opt/hearth/data
install -d -m 755 -o deploy -g deploy /opt/hearth/staging

echo "==> ensure /opt/hearth/.env with SESSION_SECRET"
if [[ ! -f /opt/hearth/.env ]]; then
    # 48 random bytes → base64 → 64 chars, well over the 32-byte minimum.
    SESSION_SECRET=$(head -c 48 /dev/urandom | base64)
    {
        echo "SESSION_SECRET=${SESSION_SECRET}"
    } > /opt/hearth/.env
    chmod 600 /opt/hearth/.env
    chown deploy:deploy /opt/hearth/.env
    echo "    generated"
else
    echo "    already present (keeping existing SESSION_SECRET)"
fi

echo "==> install systemd unit"
install -m 644 "$REPO_ROOT/deploy/hearth.service" /etc/systemd/system/hearth.service
systemctl daemon-reload
systemctl enable hearth >/dev/null
echo "    installed and enabled (service will start on first deploy)"

echo "==> install Caddy site block"
install -d -m 755 /etc/caddy/sites
install -m 644 "$REPO_ROOT/deploy/Caddyfile.hearth" /etc/caddy/sites/hearth.conf
install -d -m 755 /var/log/caddy
systemctl reload caddy
echo "    reloaded; TLS auto-provisions once DNS points api.hearth.prakashvenkat.com here"

echo "==> install sudoers rule for the deploy user"
install -m 440 "$REPO_ROOT/deploy/sudoers.d-hearth" /etc/sudoers.d/hearth
visudo -c -f /etc/sudoers.d/hearth
echo "    validated"

echo
echo "==> provisioning complete"
echo
echo "Next steps:"
echo "  1. In DigitalOcean DNS, add A record:  api.hearth.prakashvenkat.com -> this droplet's IP"
echo "  2. In GitHub, add these repo Secrets:"
echo "       DROPLET_HOST     = this droplet's IP"
echo "       DROPLET_SSH_KEY  = contents of ~/.ssh/hearth_deploy on your dev machine"
echo "       HEARTH_DOMAIN    = api.hearth.prakashvenkat.com"
echo "  3. Push a change to main (or run the 'Deploy' workflow manually)."
echo "     First deploy will scp the binary, systemctl restart, and smoke-test."
