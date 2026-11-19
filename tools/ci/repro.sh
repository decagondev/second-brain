#!/usr/bin/env bash
# SB-504: build the release artefacts twice, in two directories with separate fresh build
# caches, and fail if any digest differs (sb-release(7), Builds).
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
out="${1:-$root/build/release}"
commit="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"
flags=(-trimpath -buildvcs=false -ldflags "-s -w -buildid= -X example.com/second-brain/internal/version.Commit=$commit")
for n in a b; do
  cache="$(mktemp -d)"
  mkdir -p "$out/$n"
  ( cd "$root" && CGO_ENABLED=0 GOCACHE="$cache" go build "${flags[@]}" -o "$out/$n/sb" ./cmd/sb )
  GOCACHE="$cache" go clean -cache
done
a="$(sha256sum "$out/a/sb" | cut -d' ' -f1)"
b="$(sha256sum "$out/b/sb" | cut -d' ' -f1)"
if [ "$a" != "$b" ]; then
  echo "repro: sb differs between two builds: $a vs $b" >&2
  exit 1
fi
echo "repro: sb $a (identical in two builds)"
