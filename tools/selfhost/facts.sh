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
  for name in sb-selfhost-update.timer sb-selfhost-dbdump.timer; do
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
  [ -n "$pct" ] && d+="${d:+,}{\"path\":\"$p\",\"used_pct\":$pct}"
done
[ -n "$d" ] && parts+=("\"disks\":[$d]")

# The certificate: the company's file (SB_TLS=files), or Caddy's own (acme or internal).
if [ -f "$DIR/tls/cert.pem" ]; then
  end=$(openssl x509 -enddate -noout -in "$DIR/tls/cert.pem" 2>/dev/null | cut -d= -f2)
  [ -n "$end" ] && parts+=("\"cert_expires\":\"$(date -u -d "$end" +%Y-%m-%dT%H:%M:%SZ)\"")
elif [ -f "$DIR/Caddyfile" ]; then
  parts+=("\"cert_managed\":true")
fi

# The off-box directory: on another machine, and writable by the backup job's user.
off=$(val SB_OFFSITE)
if [ -n "$off" ] && [ -d "$off" ]; then
  tgt=$(findmnt -n -o TARGET -T "$off" 2>/dev/null || echo /)
  mounted=true; onroot=false; [ "$tgt" = / ] && { mounted=false; onroot=true; }
  w=false
  "$DOCKER" run --rm -u 65532:65532 -v "$(realpath "$off"):/o" "$HELPER" sh -c 'touch /o/.sb-probe && rm -f /o/.sb-probe' >/dev/null 2>&1 && w=true
  parts+=("\"offsite\":{\"path\":\"$off\",\"mounted\":$mounted,\"writable\":$w,\"on_root\":$onroot}")
  o=$(newest "$off/db"); [ -n "$o" ] && parts+=("\"last_offsite\":\"$(iso "$o")\"")
fi
l=$(newest "$DIR/db-dumps"); [ -n "$l" ] && parts+=("\"last_dump\":\"$(iso "$l")\"")

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
