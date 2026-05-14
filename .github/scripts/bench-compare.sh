#!/usr/bin/env bash
# bench-compare.sh — Compare benchmark output against baseline
# Usage: bench-compare.sh <current.txt> <baseline.txt>
# Exits 0 if no regression > 10%, exits 1 (warning) if regression detected.

set -euo pipefail

CURRENT="$1"
BASELINE="$2"
THRESHOLD=10  # percent

declare -A baseline_ns
declare -A current_ns

# Parse baseline: "BenchmarkX  N  total_ns ns/op  alloc_bytes B/op  allocs allocs/op"
while IFS= read -r line; do
  if [[ "$line" =~ ^Benchmark([^[:space:]]+)[[:space:]]+[0-9]+[[:space:]]+([0-9.]+)[[:space:]]ns/op ]]; then
    name="${BASH_REMATCH[1]}"
    ns="${BASH_REMATCH[2]}"
    baseline_ns["$name"]="$ns"
  fi
done < "$BASELINE"

regressions=0
while IFS= read -r line; do
  if [[ "$line" =~ ^Benchmark([^[:space:]]+)[[:space:]]+[0-9]+[[:space:]]+([0-9.]+)[[:space:]]ns/op ]]; then
    name="${BASH_REMATCH[1]}"
    ns="${BASH_REMATCH[2]}"
    current_ns["$name"]="$ns"

    if [[ -v baseline_ns["$name"] ]]; then
      old_ns="${baseline_ns[$name]}"
      # Use bc for floating-point comparison
      delta=$(printf "%.1f" "$(echo "scale=4; (($ns - $old_ns) / $old_ns) * 100" | bc -l 2>/dev/null)" 2>/dev/null || echo "0")
      if (( $(echo "$delta > $THRESHOLD" | bc -l 2>/dev/null) )); then
        echo "⚠️  REGRESSION: $name: ${old_ns}ns → ${ns}ns (+${delta}%)"
        regressions=$((regressions + 1))
      elif (( $(echo "$delta < -$THRESHOLD" | bc -l 2>/dev/null) )); then
        echo "✅ IMPROVEMENT: $name: ${old_ns}ns → ${ns}ns (${delta}%)"
      else
        echo "   STABLE: $name: ${old_ns}ns → ${ns}ns (${delta}%)"
      fi
    else
      echo "   NEW: $name: ${ns}ns (no baseline)"
    fi
  fi
done < "$CURRENT"

if [ "$regressions" -gt 0 ]; then
  echo ""
  echo "::warning::${regressions} benchmark(s) regressed by >${THRESHOLD}%"
  exit 1
fi

echo ""
echo "✅ No benchmark regressions detected (>${THRESHOLD}% threshold)"
