#!/usr/bin/env bash
# make selfhost-update (sb-selfhost(7), Upgrades; SB-1604): one upgrade of the self-hosted
# cell, the way the systemd timer in deploy/selfhost/systemd runs it. No container holds
# the Docker socket: the updater runs as a one-shot container that only verifies and
# records (state/deploy/services), and this host step does the starting.
#   1. Run the updater. A release it doesn't admit changes nothing: stop here.
#   2. Read what it recorded. Refuse a tag, an image the cell didn't attest, or an image
#      this machine doesn't hold, before touching anything.
#   3. Write build/selfhost/images.env (the digests, nothing else) and bring the
#      services up on those images.
#   4. Check each container runs exactly that image, then record it under
#      state/deploy/running, which is what the cell attests to (SB-1505).
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
DIR=build/selfhost
ENV=$DIR/images.env
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f
SERVICES=(api ingest jobs updater)
compose() {
  if [ -f "$ENV" ]; then "$DOCKER" compose --env-file "$ENV" -f deploy/selfhost/compose.yaml "$@"
  else "$DOCKER" compose -f deploy/selfhost/compose.yaml "$@"; fi
}
state() { "$DOCKER" run --rm -u 0 -v sb-selfhost_state:/s "$HELPER" sh -c "$1" | tr -d '\r'; }
die() { echo "selfhost-update: $*; nothing changed" >&2; exit 1; }

[ -f "$DIR/cell.json" ] || die "no self-hosted cell here (make selfhost first)"
if ! out=$(compose --profile update run --rm updater 2>&1); then
  printf '%s\n' "$out" >&2
  die "the updater didn't admit the release"
fi
printf '%s\n' "$out"

declare -A IMG REL
for svc in "${SERVICES[@]}"; do
  rec=$(state "cat /s/deploy/services/$svc.json 2>/dev/null || true")
  [ -n "$rec" ] || die "the updater recorded nothing for $svc"
  img=$(printf '%s' "$rec" | sed -nE 's/.*"image":"([^"]+)".*/\1/p')
  rel=$(printf '%s' "$rec" | sed -nE 's/.*"release":"([^"]+)".*/\1/p')
  [[ "$img" =~ ^[^@]+/$svc@sha256:[0-9a-f]{64}$ ]] || die "$svc's image $img isn't pinned by digest"
  key=$(printf '%s' "$img" | sha256sum | cut -d' ' -f1)
  state "test -f /s/deploy/attested/$key" >/dev/null || die "$svc's image $img isn't attested by this cell"
  id=sha256:${img##*@sha256:}
  "$DOCKER" image inspect "$id" >/dev/null 2>&1 || die "this machine doesn't hold $svc's image $id"
  IMG[$svc]=$id REL[$svc]=$rel
done

tmp=$(mktemp)
for svc in "${SERVICES[@]}"; do printf 'SB_%s_IMAGE=%s\n' "${svc^^}" "${IMG[$svc]}"; done >"$tmp"
mv "$tmp" "$ENV"
compose up -d >/dev/null

for svc in api ingest jobs; do
  got=$("$DOCKER" inspect -f '{{.Image}}' "sb-selfhost-$svc-1" | tr -d '\r')
  [ "$got" = "${IMG[$svc]}" ] || { echo "selfhost-update: $svc runs $got, not ${IMG[$svc]}" >&2; exit 1; }
done
rec=
for svc in "${SERVICES[@]}"; do
  img=local/$svc@${IMG[$svc]}
  rec+="printf '{\"image\":\"%s\",\"release\":\"%s\"}' '$img' '${REL[$svc]}' >/s/deploy/running/$svc.json; "
done
state "mkdir -p /s/deploy/running && $rec chown -R 65532:65532 /s/deploy/running && chmod -R u=rwX,go= /s/deploy/running" >/dev/null
echo "selfhost-update: running ${REL[api]} on images pinned by digest"
