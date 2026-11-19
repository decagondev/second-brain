#!/usr/bin/env sh
# SB-007: a PR that changes behaviour must change the manual.
# Usage: tools/ci/manual_diff.sh BASE_REF [LABELS]
set -eu
base="${1:-origin/main}"; labels="${2:-}"
changed=$(git diff --name-only "$base"...HEAD)
code=$(echo "$changed" | grep -E '^(cmd|internal|policy|evals/suites)/' || true)
man=$(echo "$changed" | grep -E '^manual/' || true)
if [ -n "$code" ] && [ -z "$man" ]; then
  case "$labels" in
    *refactor-no-behaviour*) echo "manual-diff: override label present; logged on the PR"; exit 0 ;;
  esac
  echo "manual-diff: behaviour change without a manual/ change (ADR-0009). Files:" >&2
  echo "$code" | sed 's/^/  /' >&2
  echo "Update the man page first, or add the label refactor-no-behaviour for a pure refactor." >&2
  exit 1
fi
echo "manual-diff: ok"
