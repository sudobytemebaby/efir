#!/usr/bin/env bash
# Checks that service and handler packages meet minimum coverage thresholds.
# Thresholds: service >= 70%, handler >= 60%.
# Requires coverage.out in the current directory.
set -euo pipefail

go tool cover -func=coverage.out | grep -E "service|handler" | awk '
{
  pct = substr($3, 1, length($3)-1)
  if ($2 ~ /service/ && pct+0 < 70) {
    print "FAIL: " $0 " (below 70%)"
    exit 1
  }
  if ($2 ~ /handler/ && pct+0 < 60) {
    print "FAIL: " $0 " (below 60%)"
    exit 1
  }
}
'

echo "✓ Coverage thresholds passed."
