#!/usr/bin/env bash
# restore.sh (sb-selfhost(7), Restore; SB-1803): the database back from a nightly dump, in
# one command, or a drill that proves the newest dump restores.
#
#   restore.sh [--dump FILE|s3:NAME]
#                              restore FILE (default: the newest dump on this box), or
#                              NAME from the off-box bucket (SB-2303; sb backup list). The
#                              dump must have been made by this cell's keys: a dump from
#                              another cell is refused before anything stops. Then the
#                              writers stop, the database is restored, they start again,
#                              and the audit chain is verified.
#   restore.sh --drill         restore the newest dump into a scratch database (a container
#                              with no network, removed after), check every table of the
#                              live database is there, and record the result in
#                              build/selfhost/drill.json (sb-selfhost-drill.timer, monthly).
#                              The cell keeps running, and the live database is only read.
#   restore.sh --from-kit KIT.sealed SHARE SHARE SHARE
#                              a new machine: rebuild the box's keys from three shares of
#                              the recovery kit (SB-1902) into build/selfhost, before
#                              install.sh; which shares were used is recorded, never a share.
#
# A dump copied off the box is sealed (NAME.dump.sealed, SB-1901): --dump opens it with the
# recovery key, build/selfhost/recovery-key.private (put back by --from-kit), or --key FILE.
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

# A new machine: the keys back from the kit, before anything else exists here.
if [ "${1:-}" = --from-kit ]; then
  shift
  [ $# -ge 4 ] || die "usage: restore.sh --from-kit KIT.sealed SHARE SHARE SHARE"
  kitfile=$1; shift
  [ ! -e "$DIR/vault" ] || die "this machine has keys already ($DIR/vault); --from-kit is for a new machine"
  mkdir -p "$DIR" && chmod 700 "$DIR"
  tmp=$(mktemp "$DIR/.kit-keys.XXXXXX"); trap 'rm -f "$tmp"' EXIT
  out=$(bash tools/selfhost/sb.sh --json recovery join --kit "$kitfile" --out "$tmp" "$@") || die "the kit didn't open (above)"
  tar -C "$DIR" -xf "$tmp"
  printf '{"at":"%s","kit":%s}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$out" >>"$DIR/kit-used.jsonl"
  shares=$(printf '%s' "$out" | sed -nE 's/.*"shares":\[([0-9,]+)\].*/\1/p')
  echo "restore: the box's keys are back in $DIR from shares ${shares//,/, }"
  echo "restore: next, install.sh CONFIG (it keeps these keys), then restore.sh --dump SB_OFFSITE/db/NAME.dump.sealed"
  echo "restore: then install.sh --recovery-kit, so the recovery key leaves this box again"
  exit 0
fi
[ -f "$ENV" ] || die "no self-hosted cell here"
# shellcheck source=tools/selfhost/pg.sh
. tools/selfhost/pg.sh
TENANT=$(val SB_TENANT)
keys() { sha256sum "$DIR/vault/api/vault/tenants/$TENANT/audit/anchor-seed" 2>/dev/null | cut -c1-16; }

MODE=restore DUMP= KEY=$DIR/recovery-key.private
while [ $# -gt 0 ]; do
  case "$1" in
  --dump) DUMP=${2:-}; shift ;;
  --key) KEY=${2:-}; shift ;;
  --drill) MODE=drill ;;
  *) die "usage: restore.sh [--dump FILE|s3:NAME [--key FILE]] | --drill | --from-kit KIT.sealed SHARE…" ;;
  esac
  shift
done
# From the company's bucket (SB-2303): the sealed dump and its record, pulled to this box.
if [[ "$DUMP" == s3:* ]]; then
  # shellcheck source=tools/selfhost/offsite.sh
  . tools/selfhost/offsite.sh
  name=${DUMP#s3:}
  [ -n "$(off_url)" ] || die "SB_OFFSITE isn't a bucket here; --dump s3:NAME reads from one"
  [[ "$name" == *.sealed ]] || die "$name: only sealed copies are in the bucket (NAME.dump.sealed)"
  mkdir -p "$DIR/db-dumps/offsite" && chmod 700 "$DIR/db-dumps" "$DIR/db-dumps/offsite"
  off_get "$name" "${name%.sealed}.json" "$DIR/db-dumps/offsite" || die "$name couldn't be pulled from $(off_url) (above)"
  DUMP=$DIR/db-dumps/offsite/$(basename "$name")
fi
[ -n "$DUMP" ] || DUMP=$(ls -1t "$DIR"/db-dumps/sb-*.dump 2>/dev/null | head -1 || true)
[ -n "$DUMP" ] && [ -f "$DUMP" ] || die "no dump to restore (systemctl start sb-selfhost-dbdump.service makes one)"
# A sealed copy from off the box opens with the recovery key, into this box's dumps.
if [[ "$DUMP" == *.sealed ]]; then
  [ -f "$KEY" ] || die "$DUMP is sealed: it opens with the recovery key (restore.sh --from-kit puts it back, or --key FILE)"
  rec=${DUMP%.sealed}.json
  mkdir -p "$DIR/db-dumps" && chmod 700 "$DIR/db-dumps"
  plain=$DIR/db-dumps/$(basename "${DUMP%.sealed}")
  bash tools/selfhost/sb.sh -q backup open --key "$KEY" --out "$plain" "$DUMP" || die "$DUMP didn't open (above)"
  [ -f "$rec" ] && cp "$rec" "$plain.json"
  DUMP=$plain
fi

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
