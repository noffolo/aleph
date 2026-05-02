# Docker Secrets Pattern — Aleph-v2

All sensitive values in the Docker Compose stack are injected via Docker Secrets (tmpfs-backed files at `/run/secrets/`) rather than plain-text environment variables.

## Architecture

```
secrets/*.txt ──► docker-compose.yml (top-level secrets:) ──► /run/secrets/<name>
                                                                    │
                        ┌───────────────────────────────────────────┘
                        │
           ┌────────────┼──────────────┐
           ▼            ▼              ▼
    entrypoint.sh   _FILE env var   direct read
    (backend,       (postgres,       (Go config.go
     sidecar)        grafana)         for KEY_ENCRYPTION_KEY)
```

Three resolution patterns are used:

| Pattern | Services | How it works |
|---------|----------|-------------|
| Entrypoint script | aleph-backend, aleph-nlp-sidecar | `docker-entrypoint.sh` reads `/run/secrets/*` → exports to env → `exec "$@"` |
| `_FILE` env var convention | aleph-db, aleph-pg-backup, grafana | Official images support `<ENV>_FILE` or `<ENV>__FILE` (Grafana double-underscore) |
| Direct file read | aleph-backend (KEY_ENCRYPTION_KEY) | `config.go` prefers `/run/secrets/key_encryption_key` over env var |

## Secrets Inventory

| Secret name | File | Consumed by | Replaces env var |
|-------------|------|-------------|-----------------|
| `key_encryption_key` | `secrets/key_encryption_key.txt` | aleph-backend | `KEY_ENCRYPTION_KEY` |
| `jwt_secret` | `secrets/jwt_secret.txt` | aleph-backend | `JWT_SECRET` |
| `postgres_dsn` | `secrets/postgres_dsn.txt` | aleph-backend | `POSTGRES_DSN` |
| `aleph_api_key_secret_backend` | `secrets/aleph_api_key_secret_backend.txt` | aleph-backend | `ALEPH_API_KEY_SECRET_BACKEND` |
| `postgres_password` | `secrets/postgres_password.txt` | aleph-db, aleph-pg-backup | `POSTGRES_PASSWORD` / `PGPASSWORD` |
| `aleph_api_key_secret` | `secrets/aleph_api_key_secret.txt` | aleph-nlp-sidecar | `ALEPH_API_KEY_SECRET` |
| `grafana_admin_password` | `secrets/grafana_admin_password.txt` | grafana | `GF_SECURITY_ADMIN_PASSWORD` |

## Setup

1. Create the `secrets/` directory and populate each file:

```bash
mkdir -p secrets

# Generate random values for each secret
openssl rand -hex 32 > secrets/key_encryption_key.txt
openssl rand -hex 32 > secrets/jwt_secret.txt

# PostgreSQL — set your actual password
echo "your_secure_postgres_password" > secrets/postgres_password.txt

# DSN with the same password
echo "postgres://postgres:your_secure_postgres_password@aleph-db:5432/aleph?sslmode=disable" > secrets/postgres_dsn.txt

# Backend API key secret
openssl rand -hex 32 > secrets/aleph_api_key_secret_backend.txt

# NLP sidecar API key
openssl rand -hex 32 > secrets/aleph_api_key_secret.txt

# Grafana admin password
echo "your_secure_grafana_password" > secrets/grafana_admin_password.txt
```

2. Set file permissions (readable only by owner):

```bash
chmod 600 secrets/*.txt
```

3. Verify the Compose config:

```bash
docker compose config
```

4. Start the stack:

```bash
docker compose up --build -d
```

## Fallback for Development

For local development without Docker Secrets, set env vars in `.env`:

```env
KEY_ENCRYPTION_KEY=<your-key>
JWT_SECRET=<your-secret>
POSTGRES_DSN=postgres://postgres:password@localhost:5432/aleph?sslmode=disable
POSTGRES_PASSWORD=password
```

The entrypoint scripts and `config.go` fall back to env vars when secret files are absent.

## Swarm Mode (Production)

In Docker Swarm, secrets are managed natively:

```bash
docker secret create key_encryption_key - < secrets/key_encryption_key.txt
docker secret create jwt_secret - < secrets/jwt_secret.txt
# ...repeat for each secret
```

Remove the `file:` directives from `docker-compose.yml` and deploy with:

```bash
docker stack deploy -c docker-compose.yml aleph
```

## Security Properties

- Secret files are mounted as **tmpfs** — never written to disk on the container
- Files are owned by root (uid 0), readable only by that user (mode 0444 by default)
- Secret values are **not** visible in `docker inspect` output (unlike env vars)
- Containers cannot write to `/run/secrets/` (read-only mount)
- `.gitignore` excludes `secrets/*.txt` from version control