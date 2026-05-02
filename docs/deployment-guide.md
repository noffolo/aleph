# Deployment Guide — Aleph-v2

> **Version:** 2.0.0 · **Last updated:** April 2026 · **Scope:** Docker Compose production deployment

This guide covers deploying Aleph-v2 using Docker Compose. For local development, see [`docs/developer-onboarding.md`](./developer-onboarding.md).

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Environment Setup](#2-environment-setup)
3. [Docker Compose Deployment](#3-docker-compose-deployment)
4. [Service Configuration](#4-service-configuration)
5. [SSL / HTTPS](#5-ssl--https)
6. [Ollama Configuration](#6-ollama-configuration)
7. [Backup and Restore](#7-backup-and-restore)
8. [Monitoring](#8-monitoring)
9. [Upgrading](#9-upgrading)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Prerequisites

You need a Linux server (or local machine) with:

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| OS | Ubuntu 22.04 / Debian 12 / macOS 14 | Ubuntu 24.04 LTS |
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Disk | 20 GB SSD | 50+ GB SSD |
| Docker | 24.x | 27.x |
| Docker Compose | 2.20 | 2.27+ |
| Git | 2.40 | 2.43+ |

For the LLM features, you also need either:
- **Ollama** running locally or on another host (recommended for privacy)
- **OpenAI API key** (optional, for cloud LLM)

---

## 2. Environment Setup

### Clone the repository

```bash
git clone <repository-url> aleph-v2
cd aleph-v2
```

### Create environment file

```bash
cp .env.example .env
```

### Required variables

Edit `.env` and set all required values:

```bash
# Mandatory — backend will refuse to start without these
KEY_ENCRYPTION_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
POSTGRES_DSN="postgres://postgres:${POSTGRES_PASSWORD}@aleph-db:5432/aleph?sslmode=disable"

# LLM provider (pick at least one)
OLLAMA_BASE_URL=http://host.docker.internal:11434
# OPENAI_API_KEY=sk-...

# CORS — update with your actual domains
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# Optional but recommended
ENV=production
LOG_LEVEL=info
RATE_LIMIT_CHAT=60
RATE_LIMIT_HEALTH=120
RATE_LIMIT_DEFAULT=100
```

**Security note:** In production, use Docker secrets instead of plain environment variables. The `docker-compose.yml` already supports:
- `/run/secrets/key_encryption_key` → `KEY_ENCRYPTION_KEY`
- `/run/secrets/postgres_password` → `POSTGRES_PASSWORD`

Create the secrets:
```bash
echo -n "$(openssl rand -hex 32)" | docker secret create key_encryption_key -
echo -n "$(openssl rand -hex 16)" | docker secret create postgres_password -
```

Then uncomment the secrets section in `docker-compose.yml`.

---

## 3. Docker Compose Deployment

### Validate configuration

```bash
docker compose config
```

This prints the resolved compose file. Check for errors before starting.

### First startup

```bash
# Build all images
docker compose build

# Start all services in background
docker compose up -d

# Watch logs
docker compose logs -f
```

### Services started

The compose file brings up 4 services:

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| `aleph-backend` | Go multi-stage build | 8080 | API server |
| `aleph-frontend` | nginx:alpine | 5174 → 80 | Static SPA |
| `aleph-db` | postgres:16 | 5432 | Metadata database |
| `aleph-nlp-sidecar` | python:3.12-slim | 8001 | NLP gRPC sidecar |

**Volumes created:**
- `aleph-pgdata` — PostgreSQL data (persistent)
- `aleph-data` — Application data, projects, raw files
- `aleph-duckdb` — DuckDB analytical database
- `./aleph_tools` — Tool code mounted read-only
- `./nlp/models` — ONNX model files

### Verify startup

Wait 30–60 seconds for all services to initialize, then check health:

```bash
# Backend readiness
curl http://localhost:8080/readyz
# Expected: 200 OK

# Backend liveness
curl http://localhost:8080/livez
# Expected: 200 OK

# Frontend
curl -I http://localhost:5174
# Expected: 200 OK

# Database
docker compose exec aleph-db pg_isready -U postgres
# Expected: accepting connections
```

### Stop and remove

```bash
docker compose down              # Stop containers, keep volumes
docker compose down -v           # Stop and delete volumes (⚠️ data loss)
```

---

## 4. Service Configuration

### Backend (`aleph-backend`)

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_ADDRESS` | `:8080` | Listen address |
| `POSTGRES_DSN` | — | PostgreSQL connection (required) |
| `DUCKDB_PATH` | `./data/aleph.duckdb` | DuckDB file path |
| `NLP_ADDR` | `aleph-nlp-sidecar:8001` | NLP sidecar gRPC address |
| `KEY_ENCRYPTION_KEY` | — | AES-256-GCM key (required) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Allowed CORS origins |
| `LOG_LEVEL` | `info` | slog level (debug/info/warn/error) |

Health checks:
- HTTP `/readyz` — returns 200 when ready, 503 during startup/drain
- HTTP `/livez` — returns 200 as long as the process is alive

### Frontend (`aleph-frontend`)

The frontend is built as a static SPA and served by nginx.

Build-time variables (passed via `docker-compose.yml`):
- `VITE_API_BASE_URL` — backend URL exposed to the browser

Runtime configuration:
- `VITE_SENTRY_DSN` — optional error monitoring

### Database (`aleph-db`)

PostgreSQL 16 with persistent volume.

Health check:
- `pg_isready -U postgres -d aleph` every 5s, 5 retries

Recommended tuning for production:
```
POSTGRES_INITDB_ARGS=--encoding=UTF-8 --locale=en_US.UTF-8
```

### NLP Sidecar (`aleph-nlp-sidecar`)

Python gRPC server with:
- Non-root user (`aleph`)
- Read-only root filesystem
- `HEALTHCHECK` via gRPC channel on `localhost:50051`
- Graceful shutdown on SIGTERM (5s grace period)

Environment:
- `ALEPH_DUCKDB_PATH` — DuckDB read-only path
- `GRPC_SERVER_ADDRESS` — gRPC listen address (`[::]:8001`)

---

## 5. SSL / HTTPS

### Option A: Reverse proxy (recommended)

Put nginx, Caddy, or Traefik in front of Aleph:

**Caddy example:**
```
yourdomain.com {
    reverse_proxy localhost:5174
}

api.yourdomain.com {
    reverse_proxy localhost:8080
}
```

**nginx example:**
```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:5174;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Update `CORS_ALLOWED_ORIGINS` in `.env`:
```bash
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://api.yourdomain.com
```

### Option B: Cloudflare Tunnel

```bash
cloudflared tunnel --no-autoupdate run --token <token>
```

Zero-config SSL with Cloudflare handling certificates.

---

## 6. Ollama Configuration

### Running Ollama in Docker (optional)

If you want Ollama as part of the compose stack, add to `docker-compose.yml`:

```yaml
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    environment:
      - OLLAMA_ORIGINS=*
```

Then update `.env`:
```bash
OLLAMA_BASE_URL=http://ollama:11434
```

### Pull models

```bash
docker compose exec ollama ollama pull llama3
docker compose exec ollama ollama pull nomic-embed-text
```

### External Ollama

If Ollama runs on the host or another machine:

```bash
# Linux — use host networking
OLLAMA_BASE_URL=http://host.docker.internal:11434

# Remote server
OLLAMA_BASE_URL=http://ollama.internal:11434
```

Ensure the Ollama host accepts connections from the Docker network (set `OLLAMA_HOST=0.0.0.0` on the Ollama side if needed).

---

## 7. Backup and Restore

### Database backup

```bash
# Backup PostgreSQL
docker compose exec aleph-db pg_dump -U postgres aleph > aleph_backup_$(date +%F).sql

# Backup DuckDB
docker compose cp aleph-backend:/app/data/aleph.duckdb ./aleph.duckdb.backup

# Backup projects data
docker compose cp aleph-backend:/app/data/projects ./projects_backup
```

### Automated backup script

Create `backup.sh`:
```bash
#!/bin/bash
set -euo pipefail

BACKUP_DIR="/backups/aleph-$(date +%F_%H%M%S)"
mkdir -p "$BACKUP_DIR"

docker compose exec -T aleph-db pg_dump -U postgres aleph > "$BACKUP_DIR/postgres.sql"
docker compose cp aleph-backend:/app/data/aleph.duckdb "$BACKUP_DIR/"
tar czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"
rm -rf "$BACKUP_DIR"

echo "Backup saved: $BACKUP_DIR.tar.gz"
```

Run via cron:
```bash
0 2 * * * /path/to/backup.sh >> /var/log/aleph-backup.log 2>&1
```

### Restore

```bash
# Stop services
docker compose down

# Restore PostgreSQL
docker compose up -d aleph-db
sleep 5
docker compose exec -T aleph-db psql -U postgres < aleph_backup_2026-04-27.sql

# Restore DuckDB
docker compose cp ./aleph.duckdb.backup aleph-backend:/app/data/aleph.duckdb

# Restart everything
docker compose up -d
```

---

## 8. Monitoring

### Prometheus metrics

The backend exposes metrics at `http://localhost:8080/metrics`:

- `aleph_request_duration_seconds` — HTTP request latency
- `aleph_sse_connections` — Active SSE connections
- `aleph_tool_health_status` — Tool health (1=healthy, 0=unhealthy)
- `aleph_chat_sessions_active` — Active chat sessions

### Health endpoints

| Endpoint | Expected | Checked by |
|----------|----------|------------|
| `GET /readyz` | 200 OK | Load balancer, Docker |
| `GET /livez` | 200 OK | Kubernetes/Docker |
| `GET /api/v1/healthz` | `{"status":"ok"}` | External monitors |
| `GET /api/v1/tools/health` | JSON array | Internal diagnostics |

### Log aggregation

Structured JSON logs from `slog`:
```json
{
  "time": "2026-04-27T10:00:00Z",
  "level": "INFO",
  "msg": "request completed",
  "method": "POST",
  "path": "/aleph.v1.QueryService/Chat",
  "duration_ms": 1240,
  "project_id": "proj_abc123"
}
```

Collect with Fluent Bit, Vector, or Promtail and ship to Loki/Elasticsearch.

### Alerting rules (Prometheus example)

```yaml
groups:
  - name: aleph
    rules:
      - alert: AlephBackendDown
        expr: up{job="aleph-backend"} == 0
        for: 1m
        labels:
          severity: critical

      - alert: AlephHighErrorRate
        expr: rate(aleph_request_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning

      - alert: AlephNLPSidecarDown
        expr: up{job="aleph-nlp"} == 0
        for: 2m
        labels:
          severity: warning
```

---

## 9. Upgrading

### Zero-downtime upgrade

1. **Pull latest code**
   ```bash
   git pull origin main
   ```

2. **Review changes**
   ```bash
   cat docs/CHANGELOG.md
   ```

3. **Backup**
   ```bash
   ./backup.sh
   ```

4. **Build new images**
   ```bash
   docker compose build --no-cache
   ```

5. **Rolling restart**
   ```bash
   docker compose up -d
   ```

   Docker Compose restarts containers with changed images only. The frontend nginx and backend Go handle this gracefully.

6. **Verify**
   ```bash
   curl http://localhost:8080/readyz
   curl http://localhost:5174
   ```

### Database migrations

If the release includes schema changes, run migrations manually:

```bash
docker compose exec aleph-backend \
  go run cmd/migrate/main.go up
```

(Replace with actual migration command if different.)

---

## 10. Troubleshooting

### Containers fail to start

Check logs:
```bash
docker compose logs --tail=50 aleph-backend
docker compose logs --tail=50 aleph-db
docker compose logs --tail=50 aleph-nlp-sidecar
```

Common causes:
- `KEY_ENCRYPTION_KEY` missing → backend exits immediately
- PostgreSQL not ready → backend retries; check `pg_isready`
- Port conflict → change ports in `docker-compose.yml`

### Cannot connect to Ollama

```bash
# From inside the backend container
docker compose exec aleph-backend wget -qO- http://host.docker.internal:11434/api/tags
```

If this fails:
1. Ollama is not running on the host
2. Docker cannot reach the host network (use `host.docker.internal` on Docker Desktop, or the host IP on Linux)
3. `OLLAMA_HOST=0.0.0.0` is not set on the Ollama side

### Frontend shows blank page

1. Check `VITE_API_BASE_URL` in `docker-compose.yml` — it must be reachable from the browser
2. Check browser console for CORS errors — update `CORS_ALLOWED_ORIGINS`
3. Check nginx logs: `docker compose logs aleph-frontend`

### High memory usage

- **Backend**: limit via Docker: `mem_limit: 1g` in `docker-compose.yml`
- **Ollama**: models are large; use smaller models (`llama3:8b` instead of `70b`)
- **NLP sidecar**: ONNX model uses ~400MB RAM

### Database connection pool exhausted

Increase pool size in `.env`:
```bash
POSTGRES_DSN=postgres://...?sslmode=disable&pool_max_conns=20
```

---

## Reference

- [`docs/developer-onboarding.md`](./developer-onboarding.md) — Local development setup
- [`docs/runbook.md`](./runbook.md) — Operational procedures
- [`docs/api-reference.md`](./api-reference.md) — API reference
