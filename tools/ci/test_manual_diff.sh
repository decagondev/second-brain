#!/usr/bin/env sh
# SB-007: prove the manual-diff rule works, in a throwaway git repo.
#   code change, no manual change      -> must FAIL
#   code change + manual change        -> must pass
#   code change + refactor label       -> must pass (override)
#   docs-only change                   -> must pass
set -eu
rule="$(cd "$(dirname "$0")" && pwd)/manual_diff.sh"
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
cd "$tmp"
git init -q -b main . && git config user.email t@example.com && git config user.name test
mkdir -p internal manual docs && echo a > internal/x.go && echo a > manual/sb.1.md && echo a > docs/d.md
git add . && git commit -qm base
check() { # name expect(0|1) labels
  git checkout -q main && git checkout -q -B "$1"; shift
  expect=$1; labels=$2; shift 2
  for f in "$@"; do date +%s%N >> "$f"; done
  git commit -qam change
  if sh "$rule" main "$labels" >/dev/null 2>&1; then got=0; else got=1; fi
  [ "$got" = "$expect" ] || { echo "manual-diff test FAILED: expected $expect got $got for $*"; exit 1; }
  echo "  ok  expect=$expect  files=$*  labels=${labels:-none}"
}
check code-only     1 ""                     internal/x.go
check code-and-man  0 ""                     internal/x.go manual/sb.1.md
check override      0 "refactor-no-behaviour" internal/x.go
check docs-only     0 ""                     docs/d.md
echo "manual-diff: rule behaves as specified"
