# Aleph-v2 Deployment Guide

> Deploying the Aleph-v2 Data OS with Docker Compose, TLS, monitoring, and rollback safety.

---

## Table of Contents

1. [Hardware Requirements](#1-hardware-requirements)
2. [Docker Compose Quick Start](#2-docker-compose-quick-start)
3. [Environment Variables](#3-environment-variables)
4. [Secrets Management](#4-secrets-management)
5. [TLS / SSL via nginx](#5-tls--ssl-via-nginx)
6. [Backup and Restore](#6-backup-and-restore)
7. [Rollback Strategy](#7-rollback-strategy)
8. [Healthcheck Endpoints](#8-healthcheck-endpoints)
9. [Monitoring](#9-monitoring)
10. [Scaling](#10-scaling)

---

## 1. Hardware Requirements

| | Minimum | Recommended |
|---|---|---|
| **RAM** | 4 GB | 8 GB |
| **CPU** | 2 cores | 4 cores |
| **Disk** | 10 GB SSD | 50 GB SSD |

The minimum spec is enough for evaluation and small teams. For production workloads with large DuckDB datasets, multiple active agents, and the Prometheus/Grafana observability stack, use the recommended spec.

---

## 2. Docker Compose Quick Start

Clone the repo and start the stack. You'll need Docker Engine >=24.x and Docker Compose >=2.20.

```bash
git clone https://github.com/noffolo/aleph.git
cd aleph

# 1. Set up environment and secrets
mkdir -p secrets
openssl rand -hex 32 > secrets/key_encryption_key.txt
openssl rand -hex 32 > secrets/jwt_secret.txt
openssl rand -base64 32 > secrets/postgres_password.txt

# Build the DSN using the same password
PGPASS=$(cat secrets/postgres_password.txt)
echo "postgres://postgres:${PGPASS}@aleph-db:5432/aleph?sslmode=disable" > secrets/postgres_dsn.txt

# Remaining secrets
openssl rand -hex 32 > secrets/aleph_api_key_secret_backend.txt
openssl rand -hex 32 > secrets/aleph_api_key_secret.txt
echo "your-grafana-admin-password" > secrets/grafana_admin_password.txt

# Protect secrets
chmod 600 secrets/*.txt

# 2. Build and start everything
docker compose up --build -d
```

Verify after ~60 seconds:

```bash
curl -sf http://localhost:8080/readyz  && echo "ready"
curl -sf http://localhost:8080/livez   && echo "alive"
curl -sf http://localhost:8080/api/v1/healthz && echo "health ok"
curl -sf http://localhost:8080/metrics && echo "metrics ok"
```

The web UI runs on `http://localhost:5173`.

> **Note:** In development, the Vite dev server runs on port 5173. Docker Compose maps to port 5174 to avoid conflicts.

---

## 3. Environment Variables

Aleph reads variables from `.env` (for local dev) or Docker Secrets (for production). The table below covers every variable Aleph uses.

### Required variables

| Variable | Description | Example |
|---|---|---|
| `KEY_ENCRYPTION_KEY` | AES-256-GCM hex key for encrypting API keys at rest | `$(openssl rand -hex 32)` |
| `JWT_SECRET` | Symmetric JWT signing key; rotation invalidates all sessions | `$(openssl rand -hex 32)` |
| `POSTGRES_DSN` | Full PostgreSQL connection string | `postgres://postgres:pass@aleph-db:5432/aleph?sslmode=disable` |
| `POSTGRES_PASSWORD` | PostgreSQL superuser password, used by the db container | `$(openssl rand -base64 32)` |

These four are injected via Docker Secrets in production. The backend refuses to start if any is missing.

### Backend configuration

| Variable | Default | Description | Required |
|---|---|---|---|
| `PORT` | `8080` | HTTP listen port | Optional |
| `SERVER_ADDRESS` | `:8080` | Full bind address (Docker Compose sets this) | Optional |
| `DATA_ROOT` | `./data/raw` | Where raw uploaded files are stored | Optional |
| `DUCKDB_PATH` | `./data/aleph.duckdb` | Analytical database file path | Optional |
| `DUCKDB_SCHEMA` | `main` | Default DuckDB schema namespace | Optional |
| `NLP_ADDR` | `localhost:8001` | Python NLP sidecar gRPC address | Optional |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama API endpoint | Optional |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Model used for vector embeddings | Optional |
| `LOG_LEVEL` | `info` | slog level: debug, info, warn, error | Optional |
| `ENV` | `development` | Environment label: development, staging, production | Optional |

### Rate limiting

| Variable | Default | Description |
|---|---|---|
| `RATE_LIMIT_CHAT` | `10` (dev) / `60` (example) | Requests per minute for chat endpoints |
| `RATE_LIMIT_HEALTH` | `100` | Requests per minute for health/readiness |
| `RATE_LIMIT_DEFAULT` | `500` | Default RPM for all other endpoints |

### Backup and retention

| Variable | Default | Description |
|---|---|---|
| `BACKUP_INTERVAL` | `24h` | How often the DuckDB backup runs |
| `BACKUP_DIR` | `./data/backups/duckdb` | Backup destination path |
| `BACKUP_KEEP` | `7` | Number of recent backups to retain |

### Limits and thresholds

| Variable | Default | Description |
|---|---|---|
| `MAX_PROJECTS` | `50` | Max active projects |
| `MAX_AGENTS_PER_PROJECT` | `20` | Max agents per project |
| `LLM_TIMEOUT_SECONDS` | `30` | Timeout for model calls |
| `SLOW_QUERY_THRESHOLD_MS` | `500` | Threshold above which queries are logged as slow |

### Cross-origin and networking

| Variable | Default | Description |
|---|---|---|
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://localhost:3000` | Comma-separated list of allowed origins |
| `MCP_SERVER_URIS` | — | Comma-separated list of MCP server URIs |

### Grafana and Prometheus

| Variable | Description | Default |
|---|---|---|
| `GRAFANA_ADMIN_USER` | Grafana login username | `admin` |
| `GF_SECURITY_ADMIN_PASSWORD` | Grafana admin password | Set via Docker Secret |

### Docker Compose service limits

| Service | CPU Limit | Memory Limit |
|---|---|---|
| `aleph-backend` | `1` | `512` MB |
| `aleph-python-sidecar` | `2` | `2` GB |
| `aleph-frontend` | `0.5` | `256` MB |
| `aleph-db` | `1` | `512` MB |
| `aleph-ollama` | `2` | `8` GB |
| `prometheus` | `0.5` | `512` MB |
| `grafana` | `0.5` | `256` MB |
| `alertmanager` | `0.25` | `128` MB |

---

## 4. Secrets Management

Aleph never reads secrets from plain environment variables in production. All sensitive values are passed as Docker Secrets, mounted at `/run/secrets/` inside each container.

### How secrets flow

```
secrets/*.txt files
       |
       v
docker-compose.yml top-level secrets:
       |
       +---> aleph-backend  /run/secrets/key_encryption_key
       +---> aleph-db       /run/secrets/postgres_password
       +---> grafana        /run/secrets/grafana_admin_password
```

The entrypoint script (`docker-entrypoint.sh`) reads each secret, exports it as an environment variable, then `exec`s the real application. This keeps the app code unchanged while hardening the secret surface.

### Setup commands

```bash
mkdir -p secrets

# Core encryption and auth
openssl rand -hex 32 > secrets/key_encryption_key.txt
openssl rand -hex 32 > secrets/jwt_secret.txt

# Database
echo "your_secure_password" > secrets/postgres_password.txt
echo "postgres://postgres:your_secure_password@aleph-db:5432/aleph?sslmode=disable" > secrets/postgres_dsn.txt

# API key secrets
openssl rand -hex 32 > secrets/aleph_api_key_secret_backend.txt
openssl rand -hex 32 > secrets/aleph_api_key_secret.txt

# Observability
echo "your_grafana_password" > secrets/grafana_admin_password.txt

chmod 600 secrets/*.txt
```

### gosecrets (alternative for bare-metal or systemd)

If you run Aleph outside Docker, use the `gosecrets` tool to store secrets in the operating system's credential store:

```bash
# Set
gosecrets set key_encryption_key "$(openssl rand -hex 32)"
gosecrets set jwt.secret "$(openssl rand -hex 32)"
gosecrets set postgres.dsn "postgres://..."
```

For development, see the secrets section in [docs/developer-onboarding.md](developer-onboarding.md). For production, use Docker Secrets.

On startup, the backend calls `LoadSecrets()`, which first checks `gosecrets`, then falls back to env vars. Set `GOSECRETS_ENV=ci` to force env-var-only mode in CI pipelines.

### Important rules

- Never commit `.env`, `secrets/*.txt`, or any file containing real credentials.
- Add `secrets/*.txt` to `.gitignore`.
- Rotate `KEY_ENCRYPTION_KEY` only after exporting and re-importing all encrypted data. Old API keys become unreadable after rotation.
- Rotate `JWT_SECRET` to force a global logout of all active sessions.

---

## 5. TLS / SSL via nginx

Place nginx in front of the Docker stack as a reverse proxy with TLS termination.

### nginx configuration

Save as `/etc/nginx/conf.d/aleph.conf`:

```nginx
# Aleph-v2 NGINX Reverse Proxy with TLS Termination

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    keepalive_timeout 65;
    gzip on;
    gzip_types text/plain application/json text/css application/javascript;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/m;
    limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s;

    upstream aleph_backend {
        server localhost:8080;
        keepalive 32;
    }

    # Redirect all HTTP to HTTPS
    server {
        listen 80;
        server_name _;
        return 301 https://$host$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.example.com;

        ssl_certificate /etc/letsencrypt/live/your-domain.example.com/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/your-domain.example.com/privkey.pem;

        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
        ssl_prefer_server_ciphers on;
        ssl_session_cache shared:SSL:10m;
        ssl_session_timeout 10m;

        # Security headers
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-Frame-Options "DENY" always;

        # Auth endpoints (rate-limited)
        location /api/v1/auth/ {
            proxy_pass http://aleph_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto https;
            limit_req zone=auth burst=3 nodelay;
        }

        # SSE / Events (WebSocket upgrade support)
        location /api/v1/events {
            proxy_pass http://aleph_backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto https;
            proxy_buffering off;
            proxy_cache off;
            proxy_read_timeout 86400s;
        }

        # General API
        location /api/ {
            proxy_pass http://aleph_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto https;
            limit_req zone=api burst=10 nodelay;
        }

        # Health check (no rate limit)
        location /healthz {
            proxy_pass http://aleph_backend/api/v1/healthz;
            proxy_set_header Host $host;
            proxy_set_header X-Forwarded-Proto https;
        }

        # Static frontend
        location / {
            proxy_pass http://localhost:5174;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto https;
        }
    }
}
```

Validate and reload:

```bash
nginx -t
systemctl reload nginx
```

### Certbot and automatic renewal

```bash
# Install certbot
sudo apt update && sudo apt install certbot python3-certbot-nginx

# Issue certificate
sudo certbot --nginx -d your-domain.example.com

# Test auto-renewal
sudo certbot renew --dry-run
```

Certbot installs a systemd timer that renews certificates automatically. Verify it is active:

```bash
systemctl list-timers | grep certbot
```

### Self-signed certificate for testing

```bash
mkdir -p /etc/nginx/ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/nginx/ssl/selfsigned.key \
  -out /etc/nginx/ssl/selfsigned.crt \
  -subj "/CN=localhost"
```

Update the `ssl_certificate` and `ssl_certificate_key` paths in `nginx.conf` accordingly.

---

## 6. Backup and Restore

### PostgreSQL backup

The compose stack includes an optional `aleph-pg-backup` service that runs `pg_dump` every 24 hours. To start it:

```bash
docker compose --profile backup up -d aleph-pg-backup
```

Backups land in `./backups/postgres/` with filenames like `aleph_pg_20260502_143052.sql.gz`.

You can also run a one-off backup manually:

```bash
docker compose exec aleph-db pg_dump -U postgres aleph > aleph_pg_$(date +%F_%H%M%S).sql
```

### PostgreSQL restore

```bash
# 1. Stop everything
docker compose down

# 2. Start PostgreSQL only
docker compose up -d aleph-db
sleep 10

# 3. Recreate the database
docker compose exec aleph-db psql -U postgres -c "DROP DATABASE IF EXISTS aleph;"
docker compose exec aleph-db psql -U postgres -c "CREATE DATABASE aleph;"

# 4. Restore from dump
docker compose exec -T aleph-db psql -U postgres -d aleph < aleph_pg_backup.sql

# 5. Restart the full stack
docker compose up -d
```

### DuckDB backup

DuckDB is a single file inside the `aleph-backend` container (`/app/aleph_registry.duckdb`). Copy it out before any major operation.

```bash
# One-off backup
docker compose cp aleph-backend:/app/aleph_registry.duckdb ./backups/aleph_$(date +%F_%H%M%S).duckdb

# Automated via cron on the host
0 2 * * * docker compose cp aleph-backend:/app/aleph_registry.duckdb /opt/backups/aleph_$(date +\%F).duckdb
```

### DuckDB restore

```bash
# 1. Stop the backend
docker compose stop aleph-backend

# 2. Start the backend (required for `docker compose cp` on some Docker configurations)
docker compose start aleph-backend

# 3. Replace the database file
docker compose cp ./backups/aleph_2026-05-02.duckdb aleph-backend:/app/aleph_registry.duckdb

# 4. Restart the backend
docker compose start aleph-backend

# 5. Verify
curl -sf http://localhost:8080/readyz && echo "ok"
```

---

## 7. Rollback Strategy

If a release introduces a bug, corrupts data, or breaks an API contract, roll back using the following steps.

### Step 1: Stop the running stack

```bash
docker compose down
```

### Step 2: Restore data from backup

```bash
# Restore PostgreSQL
docker compose up -d aleph-db
sleep 10
docker compose exec -T aleph-db psql -U postgres -d aleph < /path/to/previous_backup.sql

# Restore DuckDB
docker compose start aleph-backend
docker compose cp /path/to/previous.duckdb aleph-backend:/app/aleph_registry.duckdb
```

### Step 3: Checkout the previous stable tag

```bash
git fetch --tags
TAG=$(git tag --list --sort=-v:refname | sed -n '2p')  # previous tag
git checkout ${TAG}
```

### Step 4: Rebuild and restart

```bash
docker compose up --build -d
```

### Step 5: Verify

```bash
for endpoint in /readyz /livez /api/v1/healthz /metrics; do
  echo -n "${endpoint}: "
  curl -sf http://localhost:8080${endpoint} && echo "ok" || echo "FAIL"
done
```

### Zero-downtime rollback (advanced)

For production systems with live users, use a shadow stack:

1. Spin up the previous tag on a second host or second Docker network.
2. Point nginx upstream to the new (old) backend.
3. Switch DNS or nginx upstream weights once the shadow stack passes smoke tests.
4. Keep the broken stack running for diagnosis, then destroy it after the rollback is confirmed stable.

### Pre-deploy checklist (quick reference)

Before any production deploy, always do this:

```bash
# 1. Backup PostgreSQL
docker compose exec aleph-db pg_dump -U postgres aleph > backups/aleph_pre_deploy_$(date +%Y%m%d_%H%M%S).sql

# 2. Backup DuckDB
docker compose cp aleph-backend:/app/data/aleph.duckdb backups/aleph_pre_deploy_$(date +%Y%m%d_%H%M%S).duckdb

# 3. Verify migration status
docker compose exec aleph-db psql -U postgres -d aleph -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;"

# 4. Run any `.down.sql` rollback scripts in reverse order if a migration must be undone
for f in $(ls migrations/postgres/*.down.sql | sort -r); do
  docker compose exec aleph-db psql -U postgres -d aleph -f "$f"
done
```

---

## 8. Healthcheck Endpoints

Aleph exposes three standard Kubernetes-style probes plus a metrics endpoint.

| Endpoint | Path | Status | Purpose |
|---|---|---|---|
| **Readiness** | `GET /readyz` | `200 {"status":"ok"}` / `503 {"status":"not ready","reason":"draining"}` | Load balancer and orchestrators use this to decide whether to route traffic. Returns 503 during graceful shutdown. |
| **Liveness** | `GET /livez` | `200 {"status":"alive"}` | Kubernetes and Docker restart the container if this returns non-200. |
| **Health** | `GET /api/v1/healthz` | `200 {"status":"ok"}` | External monitoring services; unauthenticated. |
| **Metrics** | `GET /metrics` | `200` | Prometheus scrape target. See [Monitoring](#9-monitoring). |

### Usage examples

```bash
# Kubernetes readiness probe
curl -sf http://localhost:8080/readyz

# Docker HEALTHCHECK
curl -sf http://localhost:8080/livez

# Datadog / UptimeRobot
curl -sf http://localhost:8080/api/v1/healthz
```

### Docker Compose built-in health checks

The `docker-compose.yml` already defines health checks for the backend, PostgreSQL, NLP sidecar, and Ollama. You can inspect statuses with:

```bash
docker compose ps
```

---

## 9. Monitoring

### Prometheus

Prometheus is included in the compose stack and scrapes the backend every 15 seconds.

| Job | Scrape address | Path |
|---|---|---|
| `aleph-backend` | `aleph-backend:8080` | `/metrics` |
| `prometheus` | `localhost:9090` | `/metrics` |

Key metrics exposed by the backend:

| Metric | Type | Description |
|---|---|---|
| `aleph_request_duration_seconds` | Histogram | HTTP request latency |
| `aleph_sse_connections` | Gauge | Active Server-Sent Events connections |
| `aleph_tool_health_status` | Gauge | `1` if healthy, `0` if unhealthy |
| `aleph_chat_sessions_active` | Gauge | Currently active chat sessions |
| `go_memstats_heap_inuse_bytes` | Gauge | Application heap usage |
| `process_cpu_seconds_total` | Counter | Total CPU seconds consumed |

Prometheus UI runs at `http://localhost:9090`.

### Grafana

Grafana is also included. It reads dashboards from `deploy/grafana/dashboards/aleph-overview.json` and uses Prometheus as the default data source.

| Field | Value |
|---|---|
| URL | `http://localhost:3000` |
| Default user | `admin` (or whatever `GRAFANA_ADMIN_USER` is set to) |
| Password | The value in `secrets/grafana_admin_password.txt` |

Provisioning files in `deploy/grafana/provisioning/` set up the Prometheus data source automatically on first start.

### Alertmanager

Alertmanager is running on port `9093`. It is already wired to Prometheus in `prometheus.yml`.

### Prometheus alert rules example

```yaml
groups:
  - name: aleph
    rules:
      - alert: AlephBackendDown
        expr: up{job="aleph-backend"} == 0
        for: 1m
        labels:
          severity: critical

      - alert: AlephHighMemory
        expr: go_memstats_heap_inuse_bytes / (1024 * 1024) > 400
        for: 5m
        labels:
          severity: warning

      - alert: AlephNLPSidecarDown
        expr: up{job="aleph-nlp-sidecar"} == 0
        for: 2m
        labels:
          severity: warning
```

---

## 10. Scaling

### Known limit: DuckDB is single-node

DuckDB, Aleph's analytical and vector similarity engine, is an embedded database. It runs inside the Go process and stores its data in a single file on disk. This means one `aleph-backend` container is the only process that can write to the DuckDB file at any given time.

You cannot safely run two `aleph-backend` replicas pointing at the same DuckDB volume. Concurrent writes will corrupt the file.

Because of this constraint, horizontal scaling of the backend itself is limited. The recommended production pattern is to scale vertically (bigger instance with more RAM and faster SSD) rather than horizontally.

### Multi-project strategy

Aleph isolates data by project. If you need to serve multiple teams or large organizations:

1. **One stack per tenant.** Deploy a full Docker Compose stack per project or customer. Each stack has its own DuckDB, PostgreSQL, and Ollama instance. This is the safest pattern and avoids noisy-neighbor problems.

2. **Shared PostgreSQL, isolated DuckDB.** Run a single PostgreSQL cluster for metadata, but mount a separate DuckDB volume per tenant. The backend process still handles only one tenant per container, but the database infrastructure is consolidated.

3. **Read replicas (planned / advanced).** For read-heavy analytics workloads, future versions of Aleph may support copying the DuckDB file to read-only replicas. There is no official write-replication support today. Keep backup and restore tight if you experiment with manual file synchronization.

### Resource tuning for larger workloads

| Bottleneck | Fix |
|---|---|
| DuckDB queries slow | Increase backend memory limit; move DuckDB volume to NVMe SSD |
| Ollama OOM | Increase the Ollama container memory limit; use smaller models or quantize |
| PostgreSQL contention | Tune PostgreSQL `max_connections` and add `pool_max_conns` to `POSTGRES_DSN` |
| High SSE fan-out | Single backend limits broadcast fans; use a larger instance or reduce event granularity |

---

## Reference

- [AGENTS.md](../AGENTS.md) — Agent map and workflow
- [ARCHITECTURE.md](../ARCHITECTURE.md) — System design and data flow
- [SECURITY.md](../SECURITY.md) — Vulnerability reporting and security model
- [CHANGELOG.md](./CHANGELOG.md) — Release history and tag list
- [deploy/docker-secrets-readme.md](../deploy/docker-secrets-readme.md) — Deep dive into Docker Secrets
