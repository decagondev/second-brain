#!/usr/bin/env bash
# The sb command on a self-hosted box (sb-selfhost(7)): the install bundle's bin/sb, or one
# built here in a checkout with Go. The host-side steps (doctor, bundle checks, sealing,
# the recovery kit) run it, so the box needs no Go.
#   sb.sh ARGS...
set -euo pipefail
cd "$(dirname "$0")/../.."
if [ ! -x bin/sb ]; then
  command -v go >/dev/null || { echo "sb.sh: no bin/sb (run from an install bundle, or install Go)" >&2; exit 2; }
  go build -o bin/sb ./cmd/sb
fi
exec bin/sb "$@"
