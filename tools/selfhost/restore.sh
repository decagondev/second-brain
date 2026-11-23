#!/usr/bin/env bash
# restore.sh (sb-selfhost(7), Restore; SB-1803): the database back from a nightly dump, in
# one command, or a drill that proves the newest dump restores.
#
#   restore.sh [--dump FILE]   restore FILE (default: the newest dump on this box). The
#                              dump must have been made by this cell's keys: a dump from
#                              another cell is refused before anything stops. Then the
#                              writers stop, the database is restored, they start again,
#                              and the audit chain is verified.
#   restore.sh --drill         restore the newest dump into a scratch database (a container
#                              with no network, removed after), check every table of the
#                              live database is there, and record the result in
#                              build/selfhost/drill.json (sb-selfhost-drill.timer, monthly).
#                              The cell keeps running, and the live database is only read.
#
# With SB_DATABASE=external both work against the company's database (SB-1807).
#
# DOCKER names the Docker command.
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
DIR=build/selfhost
ENV=$DIR/compose.env
die() { echo "restore: $*" >&2; exit 1; }
compose() { DOCKER="$DOCKER" bash tools/selfhost/compose.sh "$@"; }
val() { sed -nE "s/^$1=//p" "$ENV" | tr -d '\r' | tail -1; }
[ -f "$ENV" ] || die "no self-hosted cell here"
# shellcheck source=tools/selfhost/pg.sh
. tools/selfhost/pg.sh
TENANT=$(val SB_TENANT)
keys() { sha256sum "$DIR/vault/api/vault/tenants/$TENANT/audit/anchor-seed" 2>/dev/null | cut -c1-16; }

MODE=restore DUMP=
while [ $# -gt 0 ]; do
  case "$1" in
  --dump) DUMP=${2:-}; shift ;;
  --drill) MODE=drill ;;
  *) die "usage: restore.sh [--dump FILE] | --drill" ;;
  esac
  shift
done
[ -n "$DUMP" ] || DUMP=$(ls -1t "$DIR"/db-dumps/sb-*.dump 2>/dev/null | head -1 || true)
[ -n "$DUMP" ] && [ -f "$DUMP" ] || die "no dump to restore (systemctl start sb-selfhost-dbdump.service makes one)"

# Whose dump is it? dbdump.sh records the tenant and a fingerprint of the cell's audit key
# beside each dump. Restoring another cell's dump under these keys would leave an audit
# chain whose anchors verify under neither.
rec=$DUMP.json
[ -f "$rec" ] || die "$DUMP has no record of the cell that made it ($rec)"
t=$(sed -nE 's/.*"tenant": *"([^"]+)".*/\1/p' "$rec"); k=$(sed -nE 's/.*"keys": *"([^"]+)".*/\1/p' "$rec")
[ "$t" = "$TENANT" ] || die "$DUMP is $t's, not $TENANT's"
[ -n "$(keys)" ] && [ "$k" = "$(keys)" ] || die "$DUMP was made under other keys (put the old build/selfhost back first; sb-selfhost(7), Restore)"

if [ "$MODE" = drill ]; then
  at=$(date -u +%Y-%m-%dT%H:%M:%SZ); ok=false; tables=0; missing=
  record() {
    printf '{"at":"%s","dump":"%s","ok":%s,"tables":%s%s}\n' "$at" "$(basename "$DUMP")" "$ok" "$tables" \
      "${missing:+,\"missing\":\"$missing\"}" >"$DIR/drill.json.tmp" && mv "$DIR/drill.json.tmp" "$DIR/drill.json"
  }
  trap 'pg_scratch_stop; record' EXIT
  q="select table_schema||'.'||table_name from information_schema.tables where table_type='BASE TABLE' and table_schema not in ('pg_catalog','information_schema') order by 1"
  pg_scratch_start sb-selfhost-drill || die "the scratch database didn't start"
  pg_scratch pg_restore --no-owner --no-privileges --exit-on-error <"$DUMP" || die "the dump $DUMP doesn't restore"
  live=$(pg psql -v ON_ERROR_STOP=1 -Atq -c "$q"); got=$(pg_scratch psql -v ON_ERROR_STOP=1 -Atq -c "$q")
  [ -n "$live" ] || die "the live database's tables couldn't be listed"
  missing=$(comm -23 <(printf '%s\n' "$live") <(printf '%s\n' "$got") | tr '\n' ' ' | sed 's/ $//')
  tables=$(printf '%s\n' "$got" | grep -c . || true)
  [ -z "$missing" ] || die "the restored dump lacks tables the live database has: $missing"
  [ "$tables" -gt 0 ] || die "the restored dump has no tables"
  ok=true
  echo "restore: drill passed: $(basename "$DUMP") restores, $tables tables"
  exit 0
fi

echo "restore: restoring $(basename "$DUMP") into $TENANT's database; the writers stop meanwhile"
compose stop api ingest jobs >/dev/null
pg pg_restore --clean --if-exists --no-owner <"$DUMP" || {
  compose up -d >/dev/null; die "pg_restore failed; the services are started again on what the database holds"
}
compose up -d >/dev/null
host=$(val SB_HOST); port=$(val SB_PORT); bind=$(val SB_BIND); [ "$bind" = 0.0.0.0 ] && bind=127.0.0.1
for _ in $(seq 1 60); do
  code=$("$CURL" -sk -o /dev/null -w '%{http_code}' --max-time 5 --resolve "$host:$port:${bind:-127.0.0.1}" "https://$host:$port/readyz" 2>/dev/null || true)
  [[ "$code" == 2* ]] && break
  sleep 2
done
[[ "${code:-}" == 2* ]] || die "the cell didn't come back ready after the restore (tools/selfhost/compose.sh logs api)"
# The audit chain, from the cell itself, as an auditor would check it.
ca=; [ -f "$DIR/ca/caddy-root.crt" ] && ca=$DIR/ca/caddy-root.crt; [ -f "$DIR/ca/company-ca.crt" ] && ca=$DIR/ca/company-ca.crt
SSL_CERT_FILE=${ca:-${SSL_CERT_FILE:-}} SB_TOKEN=$(cat "$DIR/vault/api/vault/tenants/$TENANT/control-plane") \
  bash tools/selfhost/sb.sh --tenant "$TENANT" audit verify --cell "https://$host$([ "$port" = 443 ] || echo ":$port")" ||
  die "restored, but the audit chain doesn't verify: see sb-audit(1) before letting people back in"
echo "restore: $TENANT restored from $(basename "$DUMP"); the index rebuilds from the sources if it is older (LAW-13)"
