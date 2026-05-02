# Migration Guide: v1.x to v2.0

This guide helps you upgrade from Aleph v1.x to v2.0. Read it fully before starting. Most migrations take under 30 minutes for a standard Docker Compose deployment.

---

## Before You Start

- Back up your `.env` file and your DuckDB/PostgreSQL data volumes.
- Ensure you have `openssl` available to generate a new encryption key.
- Review the [CHANGELOG](../CHANGELOG.md) for the full list of changes.

---

## Breaking Changes

### 1. KEY_ENCRYPTION_KEY is now mandatory

In v1.x, `KEY_ENCRYPTION_KEY` was optional and API keys were stored in plain text or with basic hashing. In v2.0, the system refuses to start if this variable is missing.

**What you need to do:**

```bash
# Generate a 256-bit key
openssl rand -hex 32
```

Add the result to your `.env`:

```bash
KEY_ENCRYPTION_KEY=your-generated-key-here
```

**Important:** Store this key in a password manager or secret vault. Losing it means losing access to all stored API keys.

### 2. API key hashing changed from SHA-256 to Argon2id

Existing API keys hashed with SHA-256 are automatically detected and re-hashed with Argon2id on first use. No manual action is required, but you may see a one-time performance hit during the first request after upgrade.

### 3. CSP policy hardened

Inline `<style>` tags and inline event handlers are now blocked by the Content Security Policy. If you have custom frontend plugins or injected HTML, move all styles to external CSS files.

### 4. CORS origin wildcard removed

The backend no longer accepts `*` as a CORS origin. You must list explicit origins in `CORS_ALLOWED_ORIGINS` in your `.env`.

---

## Step-by-Step Migration

### Step 1: Stop the running stack

```bash
cd /path/to/aleph-v2
docker compose down
```

### Step 2: Pull the new code

```bash
git fetch origin
git checkout v2.0.0   # or pull main if tracking branch
```

### Step 3: Update your `.env`

Add the new required variables:

```bash
# Required (new in v2.0)
KEY_ENCRYPTION_KEY=$(openssl rand -hex 32)

# CORS (no wildcards allowed)
CORS_ALLOWED_ORIGINS=http://localhost:5173,https://yourdomain.com

# Rate limiting (optional, but recommended)
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
```

Remove any deprecated variables:

- `API_KEY_HASH_ALGO` — no longer used (always Argon2id)
- `CORS_ALLOW_WILDCARD` — removed, no longer supported

### Step 4: Rebuild and restart

```bash
docker compose up --build -d
```

The Ollama service will now auto-pull `llama3` and `nomic-embed-text` on first start. This may take a few minutes.

### Step 5: Verify the migration

Run the health check suite:

```bash
# Backend
go test -race -count=1 ./...

# Frontend
cd frontend && npx vitest run && npx tsc --noEmit

# E2E
cd frontend && npx playwright test
```

Check the monitoring stack:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

### Step 6: Re-import custom tools (if applicable)

If you had custom dynamic tools registered in v1.x, re-register them through the new Genesis workflow:

1. Open the web UI.
2. Navigate to **Tools > Suggest New**.
3. Genesis will propose a sandboxed version of your tool.
4. Approve through the VetoRegistry (TTL defaults to 24 hours).

---

## Database Changes

### DuckDB

- New tables: `memory_vectors`, `repair_audit`, `genesis_proposals`, `workflow_runs`.
- Existing analytics tables are preserved; no migration script needed.

### PostgreSQL

- New migrations: `026_add_memory_namespace.sql`, `027_add_repair_log.sql`.
- Run migrations automatically on startup or manually:

```bash
go run ./cmd/migrate
```

---

## Frontend Changes

### New UI patterns

- **Terminal view** is now the default layout. Users can toggle back to dashboard mode via **View > Classic**.
- **Command palette** is accessible with `Tab` anywhere in the app.
- **SlideOver panels** replace modal dialogs for detail views.

### CSS customizations

If you had custom CSS overrides in v1.x, update them to use the new volatility layer classes:

| Old approach | New approach |
|---|---|
| Inline styles | `.vol-static` or `.vol-structural` classes |
| Hardcoded colors | CSS custom properties from `design-tokens.json` |
| Custom glassmorphism | Use `.glass-panel` class |

---

## Troubleshooting

### "FATAL: KEY_ENCRYPTION_KEY is missing"

You skipped Step 3. Add `KEY_ENCRYPTION_KEY` to `.env` and restart.

### "CORS origin not allowed"

Your `CORS_ALLOWED_ORIGINS` is either missing or contains `*`. List every origin explicitly.

### Prometheus shows "no data"

The monitoring stack is now optional in `docker-compose.yml`. Ensure the `prometheus`, `grafana`, and `alertmanager` services are not commented out.

### First request is slow after upgrade

This is the Argon2id re-hash of legacy SHA-256 API keys. It happens once per key and is expected.

---

## Rollback Plan

If something goes wrong, you can roll back to v1.x:

```bash
docker compose down
git checkout v1.x-tag-or-branch
docker compose up -d
```

Your data is safe as long as you did not delete the Docker volumes.

---

## Need Help?

- Open an issue (not for security vulnerabilities — see `SECURITY.md`).
- Read the full [technical manual](./manuale-tecnico.md) (Italian).
- Review the [API reference](./API.md).

---

*Last updated: 2026-05-02*
