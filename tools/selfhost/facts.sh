#!/usr/bin/env bash
# The box's facts (sb-selfhost(7), Doctor; SB-1801, SB-1802): one JSON document on stdout
# that `sb selfhost doctor` and the console's Box section judge (internal/core/box). It
# reads state, never secrets: service states, timers, disk use, the certificate's expiry
# date, the off-box directory's mount, backup file times and /readyz's status. A fact it
# can't gather is left out, and the verdict for it is Unknown.
#   facts.sh            DOCKER names the Docker command, CURL the curl
set -uo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
DIR=build/selfhost
ENV=$DIR/compose.env
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f
val() { sed -nE "s/^$1=//p" "$ENV" 2>/dev/null | tr -d '\r' | tail -1; }
iso() { date -u -d "@$1" +%Y-%m-%dT%H:%M:%SZ; }
newest() { # the newest dump's time in a directory, as an epoch
  ls -1t "$1"/sb-*.dump* 2>/dev/null | head -1 | xargs -r stat -c %Y 2>/dev/null
}
parts=("\"at\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"")

# Services.
if ps=$(DOCKER="$DOCKER" bash tools/selfhost/compose.sh ps -a --format '{{.Service}} {{.State}}' 2>/dev/null) && [ -n "$ps" ]; then
  s=$(printf '%s\n' "$ps" | awk 'NF==2{printf "%s{\"name\":\"%s\",\"state\":\"%s\"}", (n++?",":""), $1, $2}')
  [ "$(val SB_DATABASE)" = external ] && s+=',{"name":"postgres-external","state":"configured"}'
  parts+=("\"services\":[$s]")
fi

# Timers, where systemd runs them.
if command -v systemctl >/dev/null && systemctl list-units >/dev/null 2>&1; then
  t=
  for name in sb-selfhost-update.timer sb-selfhost-dbdump.timer sb-selfhost-drill.timer; do
    en=false; systemctl is-enabled -q "$name" 2>/dev/null && en=true
    res=$(systemctl show -p Result --value "${name%.timer}.service" 2>/dev/null)
    t+="${t:+,}{\"name\":\"$name\",\"enabled\":$en,\"result\":\"${res:-}\"}"
  done
  parts+=("\"timers\":[$t]")
fi

# Disks: this directory (keys, dumps) and Docker's data root (volumes).
d=
root=$("$DOCKER" info -f '{{.DockerRootDir}}' 2>/dev/null | tr -d '\r')
for p in "$(pwd)" ${root:+"$root"}; do
  pct=$(df -P "$p" 2>/dev/null | awk 'NR==2{gsub("%","",$5); print $5}')
  # Used and total bytes, for the disk runway (SB-3206); left out when df can't say.
  bytes=$(df -PB1 "$p" 2>/dev/null | awk 'NR==2 && $2 ~ /^[0-9]+$/ && $3 ~ /^[0-9]+$/ {printf ",\"used_bytes\":%s,\"size_bytes\":%s", $3, $2}')
  [ -n "$pct" ] && d+="${d:+,}{\"path\":\"$p\",\"used_pct\":$pct$bytes}"
done
[ -n "$d" ] && parts+=("\"disks\":[$d]")
# The samples the cell keeps of them (state/box/disk.json), so the doctor judges the same
# history as the console; none yet is no runway line.
hist=$("$DOCKER" run --rm -v sb-selfhost_state:/s:ro "$HELPER" cat /s/box/disk.json 2>/dev/null | tr -d '\r\n')
case "$hist" in "["*"]") parts+=("\"disk_history\":$hist") ;; esac

# The certificate: the company's file (SB_TLS=files), or Caddy's own (acme or internal).
if [ -f "$DIR/tls/cert.pem" ]; then
  end=$(openssl x509 -enddate -noout -in "$DIR/tls/cert.pem" 2>/dev/null | cut -d= -f2)
  [ -n "$end" ] && parts+=("\"cert_expires\":\"$(date -u -d "$end" +%Y-%m-%dT%H:%M:%SZ)\"")
elif [ -f "$DIR/Caddyfile" ]; then
  parts+=("\"cert_managed\":true")
fi

# The off-box directory: on another machine, and writable by the backup job's user. Or the
# company's bucket (SB-2303): whether it answers with the key pair, and its newest dump.
# shellcheck source=tools/selfhost/offsite.sh
. tools/selfhost/offsite.sh
off=$(val SB_OFFSITE)
s3=$(off_url)
if [ -n "$s3" ]; then
  ans=$(bash tools/selfhost/sb.sh --json backup check --to "$s3" --keys "$(off_keys)" 2>/dev/null | sed -nE 's/.*"answer":"([a-z]+)".*/\1/p')
  parts+=("\"offsite\":{\"path\":\"$s3\",\"s3\":true,\"answer\":\"${ans:-unreachable}\"}")
  if [ "$ans" = ok ]; then
    o=$(bash tools/selfhost/sb.sh backup list --from "$s3" --keys "$(off_keys)" db/ 2>/dev/null | sed -nE 's#^db/sb-([0-9-]+)T([0-9]{2})([0-9]{2})([0-9]{2})Z\.dump\.sealed$#\1T\2:\3:\4Z#p' | sort | tail -1)
    [ -n "$o" ] && parts+=("\"last_offsite\":\"$o\"")
  fi
elif [ -n "$off" ] && [ -d "$off" ]; then
  tgt=$(findmnt -n -o TARGET -T "$off" 2>/dev/null || echo /)
  mounted=true; onroot=false; [ "$tgt" = / ] && { mounted=false; onroot=true; }
  w=false
  "$DOCKER" run --rm -u 65532:65532 -v "$(realpath "$off"):/o" "$HELPER" sh -c 'touch /o/.sb-probe && rm -f /o/.sb-probe' >/dev/null 2>&1 && w=true
  parts+=("\"offsite\":{\"path\":\"$off\",\"mounted\":$mounted,\"writable\":$w,\"on_root\":$onroot}")
  o=$(newest "$off/db"); [ -n "$o" ] && parts+=("\"last_offsite\":\"$(iso "$o")\"")
fi
l=$(newest "$DIR/db-dumps"); [ -n "$l" ] && parts+=("\"last_dump\":\"$(iso "$l")\"")
# The last restore drill (SB-1803) and a hold after a rollback (SB-1805).
if [ -f "$DIR/drill.json" ]; then
  da=$(sed -nE 's/.*"at":"([^"]+)".*/\1/p' "$DIR/drill.json"); dok=$(sed -nE 's/.*"ok":(true|false).*/\1/p' "$DIR/drill.json")
  [ -n "$da" ] && parts+=("\"last_drill\":\"$da\"" "\"drill_ok\":${dok:-false}")
fi
[ -f "$DIR/held" ] && parts+=("\"held\":\"$(head -1 "$DIR/held" | tr -cd 'a-z0-9_-')\"")
# Sealing and the recovery kit (SB-1901, SB-1902): where the recovery key's private half is.
if [ -f "$DIR/recovery.pub" ]; then
  parts+=("\"sealed\":true")
  if [ -f "$DIR/recovery-key.private" ]; then rec=pending
  elif ls "$DIR"/kit/kit_*.json >/dev/null 2>&1; then rec=kit
  else rec=none; fi
  parts+=("\"recovery\":\"$rec\"")
  n=$(ls -1 "$DIR"/kit/kit_*-share-*.txt 2>/dev/null | wc -l)
  [ "$n" -gt 0 ] && parts+=("\"shares_on_box\":$n")
elif [ -f "$ENV" ]; then
  parts+=("\"sealed\":false")
fi

# /readyz through the proxy, as people reach it.
host=$(val SB_HOST); port=$(val SB_PORT); bind=$(val SB_BIND)
[ "$bind" = 0.0.0.0 ] && bind=127.0.0.1
if [ -n "$host" ] && [ -n "$port" ]; then
  code=$("$CURL" -sk -o /dev/null -w '%{http_code}' --max-time 10 --resolve "$host:$port:${bind:-127.0.0.1}" "https://$host:$port/readyz" 2>/dev/null)
  [ -n "$code" ] && [ "$code" != 000 ] && parts+=("\"ready\":$code")
fi

# Is this directory on an encrypted (LUKS) disk?
src=$(findmnt -n -o SOURCE -T "$(pwd)" 2>/dev/null)
if [ -n "$src" ] && [ -b "$src" ] && command -v lsblk >/dev/null; then
  enc=false; lsblk -s -n -o TYPE "$src" 2>/dev/null | grep -qx crypt && enc=true
  parts+=("\"encrypted\":$enc")
fi

(IFS=,; printf '{%s}\n' "${parts[*]}")
