#!/usr/bin/env bash
# The nightly database dump (sb-selfhost(7), Backup and restore; SB-1605): pg_dump of the
# cell's Postgres, kept on this machine and copied off it, fourteen nights of each. The
# backup job's write-once copies cover the audit log and approvals (sb-audit(7)); this
# covers every other record, and the index, which can also be rebuilt from the sources.
# With SB_DATABASE=external it dumps the company's database the same way (SB-1807).
#   dbdump.sh            run by sb-selfhost-dbdump.timer
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
DIR=build/selfhost
ENV=$DIR/compose.env
[ -f "$ENV" ] || { echo "dbdump: no self-hosted cell here" >&2; exit 1; }
# shellcheck source=tools/selfhost/pg.sh
. tools/selfhost/pg.sh
local_dir=$DIR/db-dumps
off=$(sed -nE 's/^SB_OFFSITE=//p' "$ENV" | tr -d '\r')
if [[ "$DOCKER" == *.exe ]] && command -v wslpath >/dev/null; then off=$(wslpath -u "$off"); fi # Docker Desktop's host path
off=$off/db
mkdir -p "$local_dir" "$off"
chmod 700 "$local_dir"
name=sb-$(date -u +%Y-%m-%dT%H%M%SZ).dump
pg pg_dump -Fc >"$local_dir/$name.tmp"
[ -s "$local_dir/$name.tmp" ] || { echo "dbdump: pg_dump wrote nothing" >&2; rm -f "$local_dir/$name.tmp"; exit 1; }
mv "$local_dir/$name.tmp" "$local_dir/$name"
chmod 600 "$local_dir/$name"
# Whose dump it is (SB-1803): the tenant and a fingerprint of the cell's audit key, so a
# restore refuses another cell's dump before it stops anything.
tenant=$(sed -nE 's/^SB_TENANT=//p' "$ENV" | tr -d '\r')
keys=$(sha256sum "$DIR/vault/api/vault/tenants/$tenant/audit/anchor-seed" | cut -c1-16)
printf '{"tenant":"%s","keys":"%s"}\n' "$tenant" "$keys" >"$local_dir/$name.json"
cp "$local_dir/$name.json" "$off/$name.json"
# Off the box, sealed to the recovery key (SB-1901): the copy here stays as it is.
if [ -f "$DIR/recovery.pub" ]; then
  bash tools/selfhost/sb.sh -q backup seal --to "$DIR/recovery.pub" --out "$off/$name.sealed" "$local_dir/$name"
  how="sealed off it"
  for plain in "$off"/sb-*.dump; do [ -e "$plain" ] && rm -f "$plain" "$plain.json"; done # copies from before sealing
else
  cp "$local_dir/$name" "$off/$name.tmp" && mv "$off/$name.tmp" "$off/$name"
  how="off it (not sealed: SB_SEAL=off)"
fi
for d in "$local_dir" "$off"; do
  ls -1t "$d"/sb-*.dump "$d"/sb-*.dump.sealed 2>/dev/null | tail -n +15 | while read -r old; do rm -f "$old" "${old%.sealed}.json"; done
done
echo "dbdump: $name on this machine and $how"
