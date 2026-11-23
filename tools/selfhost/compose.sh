#!/usr/bin/env bash
# docker compose for the self-hosted cell (sb-selfhost(7)), with the install's settings
# (build/selfhost/compose.env), its sign-in profile, and the bundled database (profile db)
# unless SB_DATABASE=external (SB-1807). Every self-hosted script and systemd unit runs
# compose through this.
#   compose.sh ARGS...      DOCKER names the Docker command
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
ENV=build/selfhost/compose.env
args=(compose -f deploy/selfhost/compose.yaml)
if [ -f "$ENV" ]; then
  args+=(--env-file "$ENV")
  for p in $(sed -nE 's/^SB_PROFILES=//p' "$ENV" | tr -d '\r'); do args+=(--profile "$p"); done
fi
grep -qx 'SB_DATABASE=external' "$ENV" 2>/dev/null || args+=(--profile db)
exec "$DOCKER" "${args[@]}" "$@"
