#!/usr/bin/env bash
# The nightly database dump (sb-selfhost(7), Backup and restore; SB-1605): pg_dump of the
# cell's Postgres, kept on this machine and copied off it, fourteen nights of each. The
# backup job's write-once copies cover the audit log and approvals (sb-audit(7)); this
# covers every other record, and the index, which can also be rebuilt from the sources.
#   dbdump.sh            run by sb-selfhost-dbdump.timer
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
DIR=build/selfhost
ENV=$DIR/compose.env
[ -f "$ENV" ] || { echo "dbdump: no self-hosted cell here" >&2; exit 1; }
local_dir=$DIR/db-dumps
off=$(sed -nE 's/^SB_OFFSITE=//p' "$ENV" | tr -d '\r')
if [[ "$DOCKER" == *.exe ]] && command -v wslpath >/dev/null; then off=$(wslpath -u "$off"); fi # Docker Desktop's host path
off=$off/db
mkdir -p "$local_dir" "$off"
chmod 700 "$local_dir"
name=sb-$(date -u +%Y-%m-%dT%H%M%SZ).dump
DOCKER="$DOCKER" bash tools/selfhost/compose.sh exec -T postgres pg_dump -U sb -d sb -Fc >"$local_dir/$name.tmp"
[ -s "$local_dir/$name.tmp" ] || { echo "dbdump: pg_dump wrote nothing" >&2; rm -f "$local_dir/$name.tmp"; exit 1; }
mv "$local_dir/$name.tmp" "$local_dir/$name"
chmod 600 "$local_dir/$name"
# Whose dump it is (SB-1803): the tenant and a fingerprint of the cell's audit key, so a
# restore refuses another cell's dump before it stops anything.
tenant=$(sed -nE 's/^SB_TENANT=//p' "$ENV" | tr -d '\r')
keys=$(sha256sum "$DIR/vault/api/vault/tenants/$tenant/audit/anchor-seed" | cut -c1-16)
printf '{"tenant":"%s","keys":"%s"}\n' "$tenant" "$keys" >"$local_dir/$name.json"
cp "$local_dir/$name" "$off/$name.tmp" && mv "$off/$name.tmp" "$off/$name"
cp "$local_dir/$name.json" "$off/$name.json"
for d in "$local_dir" "$off"; do
  ls -1t "$d"/sb-*.dump 2>/dev/null | tail -n +15 | while read -r old; do rm -f "$old" "$old.json"; done
done
echo "dbdump: $name on this machine and off it"
