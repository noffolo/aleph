#!/usr/bin/env bash
# Bundle gzip budget check
# Fails if any dist/assets/*.js exceeds 150KB gzipped.
# Run after `vite build`:  bash scripts/check-bundle-size.sh
set -euo pipefail

MAX_SIZE_KB=150
FAILED=0

for file in dist/assets/*.js; do
  if [ ! -f "$file" ]; then
    echo "No JS chunks found in dist/assets/. Run 'vite build' first."
    exit 1
  fi
  raw_size=$(wc -c < "$file")
  gz_size=$(gzip -c "$file" | wc -c)
  raw_kb=$((raw_size / 1024))
  gz_kb=$((gz_size / 1024))
  basename=$(basename "$file")
  if [ "$gz_kb" -gt "$MAX_SIZE_KB" ]; then
    echo "FAIL  ${basename}  raw=${raw_kb}KB  gz=${gz_kb}KB (limit ${MAX_SIZE_KB}KB gz)"
    FAILED=1
  else
    echo "OK    ${basename}  raw=${raw_kb}KB  gz=${gz_kb}KB"
  fi
done

echo ""
if [ "$FAILED" -eq 1 ]; then
  echo "✗ BUNDLE BUDGET FAILED: Some chunks exceed ${MAX_SIZE_KB}KB gzipped."
  exit 1
else
  echo "✓ BUNDLE BUDGET PASSED: All chunks within ${MAX_SIZE_KB}KB gzipped."
fi
