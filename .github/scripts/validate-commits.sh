#!/usr/bin/env bash
# Validates commit messages against Conventional Commits format.
# Required env vars: EVENT_NAME, PR_BASE_SHA, PR_HEAD_SHA, PUSH_BEFORE_SHA, CURRENT_SHA
set -euo pipefail

PATTERN='^(feat|fix|infra|refactor|test|docs|ci|style|chore)(\([a-z0-9-]+\))?: \S.{0,99}$'
ZERO_SHA='0000000000000000000000000000000000000000'

if [ "$EVENT_NAME" = "pull_request" ]; then
  range="$PR_BASE_SHA..$PR_HEAD_SHA"
elif [ -n "$PUSH_BEFORE_SHA" ] && [ "$PUSH_BEFORE_SHA" != "$ZERO_SHA" ]; then
  range="$PUSH_BEFORE_SHA..$CURRENT_SHA"
else
  range="$CURRENT_SHA^..$CURRENT_SHA"
fi

echo "Validating commits in range: $range"

status=0
while IFS= read -r subject; do
  [ -n "$subject" ] || continue
  if ! printf '%s\n' "$subject" | grep -qE "$PATTERN"; then
    echo "✗ Invalid commit message: $subject"
    status=1
  else
    echo "✓ $subject"
  fi
done < <(git log --no-merges --format=%s "$range")

exit "$status"
