# Aleph Data OS Bare-Metal Deployment Guide

This guide covers deploying Aleph on a fresh Ubuntu 22.04 or 24.04 server without Docker.
All core services run as native binaries or Python processes behind systemd.

---

## 1. Prerequisites

A server with:
- Ubuntu 22.04 or 24.04
- At least 2 CPU cores and 4 GB RAM
- A non-root user with sudo privileges

Update the system before starting.

```bash
sudo apt update && sudo apt upgrade -y
```

---

## 2. System User and Directories

Create a dedicated user and application directory.

```bash
sudo useradd -r -s /bin/bash -m -d /opt/aleph aleph
sudo mkdir -p /opt/aleph/data
sudo chown -R aleph:aleph /opt/aleph
```

---

## 3. Go Runtime and Backend Build

Install Go 1.22 or newer.

```bash
wget -q https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
```

Clone the repository into /opt/aleph and build the binary.

```bash
cd /opt/aleph
sudo -u aleph git clone https://github.com/ff3300/aleph-v2.git .
sudo -u aleph /usr/local/go/bin/go build -o aleph-backend .
```

---

## 4. PostgreSQL

Run the setup script as root or via sudo.

```bash
cd /opt/aleph/deploy/bare-metal
sudo ALEPH_DB_PASSWORD=your_secure_password ./setup-postgres.sh
```

This installs PostgreSQL 16, creates the aleph database and user, enables the pgcrypto and pg_trgm extensions, and updates pg_hba.conf to use scram-sha-256.

---

## 5. DuckDB

DuckDB is embedded into the Go binary via CGO. On Ubuntu, you need build essentials and a recent GCC.

```bash
sudo apt install -y build-essential gcc g++ cmake
```

The first run of aleph-backend will auto-create the DuckDB file at the path configured in ALEPH_DUCKDB_PATH.

---

## 6. Python Virtual Environment and NLP Sidecar

Install Python and create the sidecar virtual environment.

```bash
sudo apt install -y python3 python3-venv python3-pip
sudo -u aleph bash -c '
  cd /opt/aleph/nlp
  python3 -m venv venv
  ./venv/bin/pip install --upgrade pip
  ./venv/bin/pip install -r requirements.txt
'
```

Verify that grpc, numpy, pandas, and duckdb are importable.

```bash
sudo -u aleph /opt/aleph/nlp/venv/bin/python -c "import grpc, numpy, pandas, duckdb"
```

---

## 7. Environment File

Create /opt/aleph/.env owned by the aleph user. Use .env.example as a template.

```bash
sudo -u aleph cp /opt/aleph/.env.example /opt/aleph/.env
sudo -u aleph chmod 600 /opt/aleph/.env
```

Edit it and set at least:

```
ALEPH_DATABASE_URL=postgresql://aleph:your_secure_password@localhost:5432/aleph?sslmode=disable
ALEPH_DUCKDB_PATH=/opt/aleph/data/aleph.duckdb
ALEPH_NLP_GRPC_ADDRESS=localhost:8001
ALEPH_PORT=8080
ALEPH_JWT_SECRET=change_me_to_a_256_bit_random_string
```

---

## 8. nginx Reverse Proxy

Install nginx and copy the included configuration.

```bash
sudo apt install -y nginx
sudo cp /opt/aleph/deploy/bare-metal/nginx.conf /etc/nginx/sites-available/aleph
sudo ln -sf /etc/nginx/sites-available/aleph /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl restart nginx
```

The config handles SSL termination, static asset serving, WebSocket upgrade, and rate limiting.

### SSL with certbot

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.example.com
```

---

## 9. systemd Services

Install the systemd units, then enable and start them.

```bash
sudo cp /opt/aleph/deploy/bare-metal/aleph-backend.service /etc/systemd/system/
sudo cp /opt/aleph/deploy/bare-metal/aleph-nlp.service /etc/systemd/system/
sudo systemctl daemon-reload

sudo systemctl enable aleph-nlp aleph-backend
sudo systemctl start aleph-nlp
sleep 5
sudo systemctl start aleph-backend
```

View logs with journalctl.

```bash
sudo journalctl -u aleph-backend -f
sudo journalctl -u aleph-nlp -f
```

---

## 10. Firewall

Allow SSH, HTTP, and HTTPS only.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

For more hardening steps, see SECURITY.md in this directory.

---

## 11. Frontend

The frontend is a static Vite build. Build it locally or on another host and copy the dist folder to /opt/aleph/frontend/dist. nginx will serve it directly.

```bash
cd /opt/aleph/frontend
npm ci
npm run build
```

The nginx config already points to /opt/aleph/frontend/dist for static files.

---

## Directory Layout After Install

```
/opt/aleph/
  aleph-backend          compiled Go binary
  .env                   runtime secrets (chmod 600)
  data/
    aleph.duckdb         embedded analytical store
  nlp/
    venv/                Python sidecar environment
    main.py              entry point
    requirements.txt
  frontend/
    dist/                Vite production build
  deploy/
    bare-metal/          this guide and configs
```

---

## Troubleshooting

**Backend fails to start**
- Check that .env exists and ALEPH_DATABASE_URL is correct.
- Verify PostgreSQL is running: `sudo systemctl status postgresql`.
- Check journalctl logs for aleph-backend.

**NLP sidecar fails**
- Ensure the virtual environment has all requirements installed.
- Verify the DuckDB path in .env is writable by the aleph user.

**Port already in use**
- ALEPH_PORT defaults to 8080. Change it in .env if needed.
- The NLP gRPC port defaults to 8001.

---

## Next Steps

- Review SECURITY.md for fail2ban, /tmp hardening, and auto-renewal.
- Set up log rotation with logrotate.
