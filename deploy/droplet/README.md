# Hearth droplet setup

One-time setup for deploying Hearth to an existing Digital Ocean droplet that's already running Caddy + other apps on various subdomains. Fully isolated from those other apps — Hearth lives under `/opt/hearth` and `api.hearth.<domain>`.

## Prerequisites on the droplet (already true for `chrono` / 157.230.173.114)

- Ubuntu 24.04+
- Caddy installed and configured to `import /etc/caddy/sites/*` (so dropping a file there is enough)
- You can `ssh root@<droplet>` (needed for the one-time provisioning step)

## Steps

### 1. Provision the droplet (once)

As root on the droplet:

```bash
git clone https://github.com/dopatraman/hearth /tmp/hearth
cd /tmp/hearth
sudo bash deploy/droplet/provision.sh
```

This is **idempotent** — safe to re-run if something goes wrong partway. It will:

- Create a `deploy` user with the Hearth-specific SSH key pre-authorized
- Create `/opt/hearth/{bin,data,staging}` owned by `deploy`
- Install `/etc/systemd/system/hearth.service` (enabled but not started yet — no binary)
- Install `/etc/caddy/sites/hearth.conf` and reload Caddy
- Install `/etc/sudoers.d/hearth` scoping the deploy user to just the `hearth` unit

### 2. Add DNS

In DO's Networking → Domains → `prakashvenkat.com`, add:

- **Type**: A
- **Hostname**: `api.hearth`
- **Will direct to**: the droplet (157.230.173.114)
- **TTL**: 300s is fine

Caddy will auto-provision a Let's Encrypt cert once the record propagates (a minute or two).

### 3. Add GitHub Actions secrets

Repo → Settings → Secrets and variables → Actions → New repository secret. Three secrets:

| Name | Value |
|---|---|
| `DROPLET_HOST` | `157.230.173.114` (or the hostname once DNS is up) |
| `DROPLET_SSH_KEY` | The entire contents of `~/.ssh/hearth_deploy` (the **private** key) on your dev machine |
| `HEARTH_DOMAIN` | `api.hearth.prakashvenkat.com` |

Also recommended: Settings → Environments → New environment → `production`. (The workflow references `environment: production` which triggers protection rules if any — e.g., approval before deploy. Optional at MVP.)

### 4. Trigger the first deploy

Either push a commit to `main`, or run the **Deploy** workflow manually (Actions → Deploy → Run workflow).

The workflow will:

1. Cross-compile `linux/amd64`, static (`CGO_ENABLED=0`)
2. `scp` the binary to `/opt/hearth/staging/hearth.new` as the `deploy` user
3. SSH to atomic-swap into `/opt/hearth/bin/hearth` and `sudo systemctl restart hearth`
4. Poll `https://api.hearth.prakashvenkat.com/healthz` for up to 60 seconds; green if it returns `{"status":"ok"}`

## Verifying from the droplet

```bash
sudo systemctl status hearth
sudo journalctl -u hearth -f
curl -s http://127.0.0.1:8080/healthz
curl -s https://api.hearth.prakashvenkat.com/healthz
tail -f /var/log/caddy/hearth.log
```

## Rollback

```bash
# On the droplet, as deploy:
# (Keep the previous binary around if you want cheap rollback — not implemented in the deploy workflow yet.)
sudo /bin/systemctl stop hearth
# Replace /opt/hearth/bin/hearth with a previous version, then:
sudo /bin/systemctl start hearth
```

In practice, rollback = revert the commit on `main`; the deploy workflow then redeploys the prior binary.

## Uninstall

```bash
sudo systemctl stop hearth
sudo systemctl disable hearth
sudo rm /etc/systemd/system/hearth.service
sudo rm /etc/caddy/sites/hearth.conf
sudo systemctl daemon-reload
sudo systemctl reload caddy
sudo rm /etc/sudoers.d/hearth
sudo rm -rf /opt/hearth
sudo userdel -r deploy
```
