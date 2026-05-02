#!/usr/bin/env bash
# run.sh — Orchestrate all k6 load tests for Aleph-v2
#
# Usage:
#   chmod +x deploy/load-tests/run.sh
#   ./deploy/load-tests/run.sh                    # run all tests with defaults
#   ./deploy/load-tests/run.sh --quick             # run only health + query
#   ./deploy/load-tests/run.sh --vus 200           # override VUs for all tests
#   ./deploy/load-tests/run.sh --duration 60s      # override duration
#   ./deploy/load-tests/run.sh --url http://prod:8080
#
# Env vars (overridable via --flags or env):
#   BASE_URL   — target URL (default: http://localhost:8080)
#   DURATION   — sustained load duration (default: 30s)
#   VUS        — target virtual users (default: 500 for health/query, 100 for chat)
#   API_KEY    — X-Aleph-Api-Key header (default: test-key)
#   PROJECT_ID — project ID for query/chat tests (default: default)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_URL="${BASE_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
VUS="${VUS:-500}"
API_KEY="${API_KEY:-test-key}"
PROJECT_ID="${PROJECT_ID:-default}"
QUICK=false

# Parse CLI flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    --quick)    QUICK=true; shift ;;
    --vus)      VUS="$2";   shift 2 ;;
    --duration) DURATION="$2"; shift 2 ;;
    --url)      BASE_URL="$2"; shift 2 ;;
    --api-key)  API_KEY="$2"; shift 2 ;;
    --project)  PROJECT_ID="$2"; shift 2 ;;
    --help)
      echo "Usage: $0 [--quick] [--vus N] [--duration DUR] [--url URL] [--api-key KEY] [--project ID]"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

echo "============================================"
echo "  Aleph-v2 Load Tests"
echo "============================================"
echo "  Target:     ${BASE_URL}"
echo "  Duration:   ${DURATION}"
echo "  VUs:        ${VUS}"
echo "  API Key:    ${API_KEY:0:8}..."
echo "  Project:    ${PROJECT_ID}"
echo "--------------------------------------------"

run_test() {
  local name="$1"
  local script="$2"
  shift 2

  echo ""
  echo "--- Running: ${name} ---"
  K6_STATSD_ENABLE=false \
  BASE_URL="${BASE_URL}" \
  DURATION="${DURATION}" \
  VUS="${VUS}" \
  API_KEY="${API_KEY}" \
  PROJECT_ID="${PROJECT_ID}" \
  k6 run "$@" "${script}"
  echo "--- Finished: ${name} ---"
}

# 1. Health check (baseline, no auth)
run_test "Health" "${SCRIPT_DIR}/health.js"

# 2. Query (unary RPC, authenticated)
run_test "Query" "${SCRIPT_DIR}/query.js"

# 3. Chat (streaming RPC, authenticated) — skip in quick mode
if [ "${QUICK}" != true ]; then
  # Chat uses a lower VUS default since streaming is heavier
  VUS_CHAT=$(( VUS > 100 ? 100 : VUS ))
  VUS="${VUS_CHAT}" run_test "Chat" "${SCRIPT_DIR}/chat.js"
fi

echo ""
echo "============================================"
echo "  All load tests complete!"
echo "============================================"
