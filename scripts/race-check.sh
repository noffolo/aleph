#!/usr/bin/env bash
# =============================================================================
# race-check.sh — Go race detector sweep for Aleph-v2
#
# Runs `go test -race -count=1` against race-prone packages with a 5-minute
# timeout per package. Collects results and exits 0 only if ALL pass.
#
# Usage:
#   chmod +x scripts/race-check.sh
#   ./scripts/race-check.sh
#
# Packages tested:
#   1. internal/mcp/...         — MCP discovery engine, SSRF validation
#   2. internal/concurrency/... — safego, goroutine patterns
#   3. internal/ingestion/...   — ingestion engine, pipeline
#   4. internal/app/...         — SSE broker, notification service
#
# Idempotent — safe to run multiple times. No side effects on the repo.
# =============================================================================

set -euo pipefail

readonly TIMEOUT_SEC=300  # 5 minutes
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

PASSES=()
FAILS=()

# ---------------------------------------------------------------------------
# run_package <label> <package_path>
# ---------------------------------------------------------------------------
run_package() {
  local label="$1"
  local pkg_path="$2"

  printf "%b  Running: %s%b\n" "$YELLOW" "$label" "$NC"

  if timeout "$TIMEOUT_SEC" go test -race -count=1 "$pkg_path" 2>&1; then
    printf "%b  PASS: %s%b\n" "$GREEN" "$label" "$NC"
    PASSES+=("$label")
  else
    local rc=$?
    printf "%b  FAIL: %s (exit code %d)%b\n" "$RED" "$label" "$rc" "$NC"
    FAILS+=("$label")
  fi

  echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo ""
printf "%b=== Aleph-v2 Race Detector Sweep ===%b\n" "$YELLOW" "$NC"
echo ""

run_package "internal/mcp"          "./internal/mcp/..."
run_package "internal/concurrency"  "./internal/concurrency/..."
run_package "internal/ingestion"    "./internal/ingestion/..."
run_package "internal/app"          "./internal/app/..."

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
printf "%b=== Results ===%b\n" "$YELLOW" "$NC"

if [[ ${#PASSES[@]} -gt 0 ]]; then
  printf "%b  PASSED (%d):%b\n" "$GREEN" "${#PASSES[@]}" "$NC"
  for p in "${PASSES[@]}"; do
    printf "    - %s\n" "$p"
  done
fi

if [[ ${#FAILS[@]} -gt 0 ]]; then
  printf "%b  FAILED (%d):%b\n" "$RED" "${#FAILS[@]}" "$NC"
  for f in "${FAILS[@]}"; do
    printf "    - %s\n" "$f"
  done
fi

echo ""

if [[ ${#FAILS[@]} -eq 0 ]]; then
  printf "%bAll %d packages passed — no races detected.%b\n" "$GREEN" "${#PASSES[@]}" "$NC"
  exit 0
else
  printf "%b%d package(s) failed — races detected.%b\n" "$RED" "${#FAILS[@]}" "$NC"
  exit 1
fi
