# Security Hardening Checklist: Bare-Metal Deployment

This document lists hardening steps meant to be applied after completing the steps in README.md. Every item is optional to your risk model, but skipping any should be an intentional decision.

---

## Firewall (ufw)

Default deny everything except what you explicitly allow.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

If your server sits behind an external firewall, you may skip ufw and instead rely on cloud-provider rules. Document this choice.

---

## fail2ban

Install and configure fail2ban to protect SSH and nginx from brute-force attempts.

```bash
sudo apt install -y fail2ban
```

Create a local jail configuration.

```bash
sudo tee /etc/fail2ban/jail.local ><<'EOF'
[DEFAULT]
bantime  = 3600
findtime = 600
maxretry = 5
backend  = systemd

[sshd]
enabled = true
port    = ssh
filter  = sshd
logpath = /var/log/auth.log

[nginx-http-auth]
enabled = true
filter  = nginx-http-auth
port    = http,https
logpath = /var/log/nginx/error.log
EOF
```

Restart the service.

```bash
sudo systemctl restart fail2ban
sudo fail2ban-client status
```

---

## certbot Auto-Renewal

-certbot installed via the nginx plugin attempts renewal automatically through a systemd timer, but verify the timer is active.

```bash
sudo systemctl status certbot.timer
sudo certbot renew --dry-run
```

If certbot is not using the systemd timer on your distribution, add a root cron job.

```bash
sudo tee /etc/cron.d/certbot-renew ><<'EOF'
0 3 * * * root certbot renew --quiet --deploy-hook "systemctl reload nginx"
EOF
```

---

## Non-Root Service User

The systemd services already run as the `aleph` user. Ensure that user cannot log in interactively or gain elevated privileges.

```bash
sudo usermod -L aleph
sudo passwd -l aleph
```

Confirm that /opt/aleph is owned by aleph:aleph and that your non-root deployment user is the only other user who can read the .env file.

```bash
ls -ld /opt/aleph
ls -l /opt/aleph/.env
```

Revoke group- and world-read on .env.

```bash
sudo chmod 600 /opt/aleph/.env
```

---

## /tmp Hardening

If /tmp is not already mounted as a separate filesystem, consider mounting it with restrictive flags.

```bash
sudo tee -a /etc/fstab ><<'EOF'
tmpfs /tmp tmpfs defaults,nosuid,nodev,noexec,relatime,size=2G 0 0
EOF
```

Re-mount or restart to apply.

```bash
sudo mount -o remount /tmp
```

Alternatively, if you use a dedicated partition, update its mount options in /etc/fstab to include nosuid, nodev, noexec.

---

## SSH Hardening

Edit /etc/ssh/sshd_config or create a drop-in file at /etc/ssh/sshd_config.d/99-hardening.conf.

```
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
MaxAuthTries 3
ClientAliveInterval 300
ClientAliveCountMax 2
AllowUsers your-deploy-user
```

Restart SSH.

```bash
sudo systemctl restart sshd
```

---

## Log Retention

Install and configure logrotate for Aleph service logs. journald handles short-term storage, but long-term retention should be explicit.

```bash
sudo tee /etc/logrotate.d/aleph ><<'EOF'
/var/log/aleph/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 aleph aleph
    sharedscripts
    postrotate
        /bin/kill -HUP $(cat /run/systemd/journald.pid 2>/dev/null) > /dev/null 2>&1 || true
    endscript
}
EOF
```

---

## Kernel and Package Updates

Enable automatic security updates.

```bash
sudo apt install -y unattended-upgrades
sudo dpkg-reconfigure unattended-upgrades
```

Reboot after kernel upgrades and validate that all services come back up correctly.

---

## Ports Summary

| Protocol | Port | Source     | Purpose                 |
|----------|------|------------|-------------------------|
| TCP      | 22   | Your IP    | SSH                     |
| TCP      | 80   | Any        | HTTP redirect to HTTPS  |
| TCP      | 443  | Any        | HTTPS / WebSocket       |
| TCP      | 5432 | 127.0.0.1  | PostgreSQL (local only) |
| TCP      | 8080 | 127.0.0.1  | Go backend (local only) |
| TCP      | 8001 | 127.0.0.1  | NLP gRPC (local only)   |

No other ports should be reachable from the public internet.

---

## Incident Response

If you suspect a breach:

1. Snapshot the machine and detach it from the network.
2. Preserve /var/log and journald output.
3. Rotate all secrets in /opt/aleph/.env.
4. Regenerate TLS certificates and revoke the old ones.
5. Review PostgreSQL logs for unexpected queries or logins.
