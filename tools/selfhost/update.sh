#!/usr/bin/env bash
# make selfhost-update (sb-selfhost(7), Upgrades; SB-1604): one upgrade of the self-hosted
# cell, the way the systemd timer in deploy/selfhost/systemd runs it. No container holds
# the Docker socket: the updater runs as a one-shot container that only verifies and
# records (state/deploy/services), and this host step does the starting.
#   1. Run the updater. A release it doesn't admit changes nothing: stop here.
#   2. Read what it recorded. Refuse a tag, an image the cell didn't attest, or an image
#      this machine doesn't hold, before touching anything.
#   3. Write the digests into build/selfhost/compose.env (SB_*_IMAGE) and bring the
#      services up on those images. An image from a registry is pulled by its digest.
#   4. Check each container runs exactly that image, then record it under
#      state/deploy/running, which is what the cell attests to (SB-1505).
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
DIR=build/selfhost
ENV=$DIR/compose.env
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f
SERVICES=(api ingest jobs updater)
compose() { DOCKER="$DOCKER" bash tools/selfhost/compose.sh "$@"; }
state() { "$DOCKER" run --rm -u 0 -v sb-selfhost_state:/s "$HELPER" sh -c "$1" | tr -d '\r'; }
die() { echo "selfhost-update: $*; nothing changed" >&2; exit 1; }

[ -f "$DIR/cell.json" ] && [ -f "$ENV" ] || die "no self-hosted cell here (make selfhost first)"
if ! out=$(compose --profile update run --rm updater 2>&1); then
  printf '%s\n' "$out" >&2
  die "the updater didn't admit the release"
fi
printf '%s\n' "$out"

declare -A IMG REF REL
for svc in "${SERVICES[@]}"; do
  rec=$(state "cat /s/deploy/services/$svc.json 2>/dev/null || true")
  [ -n "$rec" ] || die "the updater recorded nothing for $svc"
  img=$(printf '%s' "$rec" | sed -nE 's/.*"image":"([^"]+)".*/\1/p')
  rel=$(printf '%s' "$rec" | sed -nE 's/.*"release":"([^"]+)".*/\1/p')
  [[ "$img" =~ ^[^@]+/$svc@sha256:[0-9a-f]{64}$ ]] || die "$svc's image $img isn't pinned by digest"
  key=$(printf '%s' "$img" | sha256sum | cut -d' ' -f1)
  state "test -f /s/deploy/attested/$key" >/dev/null || die "$svc's image $img isn't attested by this cell"
  if [[ "$img" == local/* ]]; then
    # A local release names images by their id, which this machine must already hold.
    id=sha256:${img##*@sha256:}
    "$DOCKER" image inspect "$id" >/dev/null 2>&1 || die "this machine doesn't hold $svc's image $id"
  else
    # A registry image is pulled by its digest, which Docker checks against the content.
    "$DOCKER" image inspect "$img" >/dev/null 2>&1 || "$DOCKER" pull -q "$img" >/dev/null || die "$svc's image $img can't be pulled"
    id=$("$DOCKER" image inspect -f '{{.Id}}' "$img" | tr -d '\r')
  fi
  IMG[$svc]=$id REF[$svc]=$img REL[$svc]=$rel
done

tmp=$(mktemp)
{
  grep -vE '^SB_[A-Z]+_IMAGE=' "$ENV" || true
  for svc in "${SERVICES[@]}"; do printf 'SB_%s_IMAGE=%s\n' "${svc^^}" "${IMG[$svc]}"; done
} >"$tmp"
mv "$tmp" "$ENV"
compose up -d >/dev/null

for svc in api ingest jobs; do
  got=$("$DOCKER" inspect -f '{{.Image}}' "sb-selfhost-$svc-1" | tr -d '\r')
  [ "$got" = "${IMG[$svc]}" ] || { echo "selfhost-update: $svc runs $got, not ${IMG[$svc]}" >&2; exit 1; }
done
rec=
for svc in "${SERVICES[@]}"; do
  img=${REF[$svc]}
  rec+="printf '{\"image\":\"%s\",\"release\":\"%s\"}' '$img' '${REL[$svc]}' >/s/deploy/running/$svc.json; "
done
state "mkdir -p /s/deploy/running && $rec chown -R 65532:65532 /s/deploy/running && chmod -R u=rwX,go= /s/deploy/running" >/dev/null
echo "selfhost-update: running ${REL[api]} on images pinned by digest"
