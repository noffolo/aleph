#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
fail() { echo -e "${RED}[FAIL]${NC} $*" >&2; exit 1; }

# required vars
REQUIRED=(POSTGRES_DSN KEY_ENCRYPTION_KEY JWT_SECRET)
for v in "${REQUIRED[@]}"; do
    val="${!v:-}"
    [ -n "$val" ] || fail "missing env var: $v"
    echo -e "${GREEN}[ok]${NC} $v"
done

# optional checks
[ -n "${POSTGRES_PASSWORD:-}" ] || echo "[warn] POSTGRES_PASSWORD not set — will use Docker secret"
[ -n "${AUTH_SECRET:-}" ]       || echo "[warn] AUTH_SECRET not set — will use default"

echo "[ok] env validation passed"
