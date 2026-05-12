#!/bin/bash
# post-create.sh — Aleph-v2 Dev Container Initialization
# Runs once after the container is created.
set -euo pipefail

cd /workspace

echo "==> Aleph-v2 Dev Container Setup"
echo "    Go: $(go version)"
echo "    Node: $(node --version)"
echo "    npm: $(npm --version)"
echo "    Python: $(python3 --version)"
echo "    Docker: $(docker --version 2>/dev/null || echo 'not available')"
echo ""

# ── 1. Environment ──────────────────────────────────────────────
if [ ! -f .env ]; then
    cp .env.example .env
    echo "[OK] Created .env from .env.example"
fi

# ── 2. Dev secrets for docker-compose services ──────────────────
mkdir -p secrets

# Generate deterministic dev secrets so docker-compose services work
generate_secret() {
    local file="$1"
    local value="$2"
    if [ ! -f "secrets/${file}.txt" ]; then
        echo -n "$value" > "secrets/${file}.txt"
        echo "[OK] Created secrets/${file}.txt"
    fi
}

generate_secret "key_encryption_key"          "$(openssl rand -hex 32)"
generate_secret "jwt_secret"                   "$(openssl rand -hex 32)"
generate_secret "postgres_password"            "devpassword"
generate_secret "postgres_dsn"                 "postgres://postgres:devpassword@aleph-db:5432/aleph?sslmode=disable"
generate_secret "aleph_api_key_secret"         "dev-api-key-$(openssl rand -hex 8)"
generate_secret "aleph_api_key_secret_backend" "dev-api-key-$(openssl rand -hex 8)"
generate_secret "grafana_admin_password"       "$(openssl rand -hex 16)"

# ── 3. Go dependencies ──────────────────────────────────────────
echo ""
echo "==> Go modules"
go mod download 2>&1 | tail -2 || echo "[WARN] go mod download failed (will retry on build)"

# ── 4. Frontend dependencies ────────────────────────────────────
echo ""
echo "==> Frontend (npm ci)"
cd /workspace/frontend
npm ci 2>&1 | tail -3
cd /workspace

# ── 5. Python NLP environment ───────────────────────────────────
echo ""
echo "==> NLP Python environment"
cd /workspace/nlp
if [ ! -d .venv ]; then
    python3 -m venv .venv
fi
# shellcheck disable=SC1091
source .venv/bin/activate
pip install --upgrade pip -q 2>&1 | tail -1 || true
pip install -r requirements.txt -q 2>&1 | tail -1 || echo "[WARN] pip install failed (check nlp/requirements.txt)"
cd /workspace

# ── 6. Data directories ─────────────────────────────────────────
mkdir -p data/raw data/ontologies data/projects data/backups/duckdb
echo ""
echo "[OK] Data directories created"

# ── 7. Git hooks ────────────────────────────────────────────────
if command -v pre-commit &>/dev/null; then
    pre-commit install 2>/dev/null || true
fi

# ── 8. Verify tooling ───────────────────────────────────────────
echo ""
echo "==> Verification"
go vet ./... 2>&1 | tail -1 || echo "[WARN] go vet has issues (may need deps)"
cd /workspace/frontend && npx tsc --noEmit 2>&1 | tail -3 || echo "[WARN] tsc has type errors (may need generated protos)"
cd /workspace

echo ""
echo "============================================"
echo "  Aleph-v2 Dev Container Ready!"
echo ""
echo "  Quick start:"
echo "    make run          — build + run Go backend"
echo "    air               — hot-reload Go backend"
echo "    make frontend-dev  — Vite dev server :5173"
echo "    make test          — run all tests"
echo "    make test-frontend — frontend tests"
echo "    make test-e2e      — Playwright E2E tests"
echo "============================================"
