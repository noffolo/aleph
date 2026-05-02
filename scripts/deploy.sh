#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
log()  { echo -e "${GREEN}[deploy]${NC} $*"; }
warn() { echo -e "${YELLOW}[warn]${NC} $*"; }
err()  { echo -e "${RED}[error]${NC} $*" >&2; }

ENV=${1:-production}
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$SCRIPT_DIR")"
cd "$ROOT"

log "deploying aleph-v2 (env=$ENV)"

# ---- validate env ----
"$SCRIPT_DIR/validate-env.sh"

# ---- pull images ----
log "pulling images..."
docker compose pull 2>/dev/null || warn "pull skipped (local build)"

# ---- generate secrets if missing ----
if [ ! -f secrets/key_encryption_key.txt ]; then
    log "generating key_encryption_key..."
    openssl rand -hex 32 > secrets/key_encryption_key.txt
    chmod 600 secrets/key_encryption_key.txt
fi
if [ ! -f secrets/postgres_password.txt ]; then
    log "generating postgres_password..."
    openssl rand -base64 32 > secrets/postgres_password.txt
    chmod 600 secrets/postgres_password.txt
fi

# ---- start services ----
log "starting services..."
docker compose up -d --build

# ---- wait for health ----
log "waiting for healthy state..."
for svc in aleph-backend aleph-db; do
    for i in $(seq 1 30); do
        status=$(docker compose ps --format json "$svc" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('Health',''))" 2>/dev/null || echo "")
        if [ "$status" = "healthy" ]; then
            log "  $svc: healthy"
            break
        fi
        sleep 2
    done
done

# ---- verify ----
log "smoke test..."
curl -sf http://localhost:8080/api/v1/healthz >/dev/null && log "  healthz: ok" || err "  healthz: FAIL"

log "deploy complete. services running."
docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
