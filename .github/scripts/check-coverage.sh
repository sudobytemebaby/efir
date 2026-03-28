#!/usr/bin/env bash
# Checks that service and handler packages meet minimum coverage thresholds.
# Thresholds: service >= 70%, handler >= 60%.
# Requires coverage.out in the current directory.
set -euo pipefail

go tool cover -func=coverage.out | awk '
/^total:/ { next }
{
  pct = substr($NF, 1, length($NF)-1) + 0
  if ($1 ~ /\/service\// && pct < 70) {
    printf "FAIL: %s (%.1f%% < 70%%)\n", $0, pct
    failed = 1
  }
  if ($1 ~ /\/handler\// && pct < 60) {
    printf "FAIL: %s (%.1f%% < 60%%)\n", $0, pct
    failed = 1
  }
}
END { exit (failed ? 1 : 0) }
'

echo "✓ Coverage thresholds passed."
