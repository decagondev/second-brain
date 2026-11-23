#!/usr/bin/env bash
# The sb command on a self-hosted box (sb-selfhost(7)): the install bundle's bin/sb, or one
# built here in a checkout with Go. The host-side steps (doctor, bundle checks, sealing,
# the recovery kit) run it, so the box needs no Go.
#   sb.sh ARGS...
set -euo pipefail
cd "$(dirname "$0")/../.."
# In a checkout with Go, built fresh each time (the build cache makes it quick), so it never
# lags the scripts; an install bundle carries its own.
if [ -d cmd/sb ] && command -v go >/dev/null; then
  go build -o bin/sb ./cmd/sb
elif [ ! -x bin/sb ]; then
  echo "sb.sh: no bin/sb (run from an install bundle, or install Go)" >&2; exit 2
fi
exec bin/sb "$@"
