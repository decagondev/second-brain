#!/usr/bin/env bash
# make selfhost-update (sb-selfhost(7), Upgrades; SB-1604): one upgrade of the self-hosted
# cell, the way the systemd timer in deploy/selfhost/systemd runs it. No container holds
# the Docker socket: the updater runs as a one-shot container that only verifies and
# records (state/deploy/services), and this host step does the starting.
#   0. Record the box's facts for the console (SB-1802).
#   1. Run the updater. A release it doesn't admit changes nothing: stop here.
#   2. Read what it recorded. Refuse a tag, an image the cell didn't attest, or an image
#      this machine doesn't hold, before touching anything.
#   3. Write the digests into build/selfhost/compose.env (SB_*_IMAGE) and bring the
#      services up on those images. An image from a registry is pulled by its digest.
#   4. Check each container runs exactly that image, then record it under
#      state/deploy/running, which is what the cell attests to (SB-1505).
#
#   update.sh                  one upgrade (the timer's run); does nothing while held
#   update.sh --bundle FILE    an offline upgrade from a bundle (SB-1804): its release is
#                              admitted and its images checked by the installed sb before
#                              any of them is loaded, then the upgrade runs as above
#   update.sh --rollback       start the previous admitted release's digests and hold (SB-1805)
#   update.sh --resume         lift the hold; the next run upgrades again
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
# The configuration and its secrets, checked before any restart (SB-2604): a typo, a missing
# secret or one others can read stops here. Files an older install wrote are made owner-only once.
checkconfig() {
  if [ ! -f "$DIR/.vault-private" ]; then chmod -R go= "$DIR/vault" 2>/dev/null || true; : >"$DIR/.vault-private"; fi
  bash tools/selfhost/sb.sh cell check-config "$DIR/cell.json" --vault "$DIR/vault/api" --vault "$DIR/vault/jobs" >/dev/null ||
    die "the configuration or its secrets are wrong (above)"
}
MODE=update BUNDLE=
case "${1:-}" in
"") ;;
--bundle) MODE=bundle BUNDLE=${2:-}; [ -f "$BUNDLE" ] || die "--bundle needs the bundle file" ;;
--rollback) MODE=rollback ;;
--resume) rm -f "$DIR/held"; echo "selfhost-update: upgrades resume at the next run"; exit 0 ;;
*) die "usage: update.sh [--bundle FILE|--rollback|--resume]" ;;
esac

[ -f "$DIR/cell.json" ] && [ -f "$ENV" ] || die "no self-hosted cell here (make selfhost first)"
checkconfig
# The box's facts for the console's Box section (SB-1802), first, so a refused release
# still leaves them fresh. Failing to record them changes nothing else.
bash tools/selfhost/facts.sh | "$DOCKER" run --rm -i -u 0 -v sb-selfhost_state:/s "$HELPER" sh -c \
  'mkdir -p /s/host && cat >/s/host/facts.tmp && mv /s/host/facts.tmp /s/host/facts.json && chown -R 65532:65532 /s/host && chmod -R u=rwX,go= /s/host' \
  || echo "selfhost-update: couldn't record the box's facts" >&2

# Held after a rollback (SB-1805): the timer's runs change nothing until --resume.
if [ "$MODE" = update ] && [ -f "$DIR/held" ]; then
  echo "selfhost-update: held at $(cat "$DIR/held") after a rollback; update.sh --resume lifts it"
  exit 0
fi

# An offline bundle (SB-1804): nothing from it reaches Docker until the installed sb has
# admitted its release and matched every image archive to the image the release names.
if [ "$MODE" = bundle ]; then
  ST=$DIR/staging; trap 'rm -rf "$ST"' EXIT
  rm -rf "$ST" && mkdir -p "$ST" && chmod 700 "$ST"
  tar -xzf "$BUNDLE" -C "$ST" --no-same-owner --no-same-permissions || die "the bundle can't be unpacked"
  root=$(find "$ST" -mindepth 1 -maxdepth 1 -type d | head -1)
  [ -d "$root/releases" ] || die "the bundle carries no release (make selfhost-bundle with SB_BUNDLE_RELEASES)"
  trusted=$(sed -nE 's/.*"trusted": *"([^"]+)".*/\1/p' "$DIR/cell.json" | head -1)
  ident=$(sed -nE 's/.*"cosign_identity": *"([^"]+)".*/\1/p' "$DIR/cell.json" | head -1)
  if [ -n "$trusted" ]; then key=(--trusted "$trusted"); else key=(--cosign-identity "$ident"); fi
  loads=$(bash tools/selfhost/sb.sh selfhost verify-bundle "$root" "${key[@]}") || die "the bundle isn't the release this cell trusts"
  while read -r f; do [ -n "$f" ] && "$DOCKER" load -q -i "$f" >/dev/null; done <<<"$loads"
  cp -a "$root/releases/." "$DIR/releases/"
  "$DOCKER" run --rm -u 0 -v sb-selfhost_releases:/dst -v "$(realpath "$root/releases"):/src:ro" "$HELPER" \
    sh -c 'cp -a /src/. /dst/ && chown -R 65532:65532 /dst && chmod -R u=rwX,go= /dst' >/dev/null
  rm -rf "$ST"
  echo "selfhost-update: the bundle's release and $(grep -c . <<<"$loads") images checked and loaded"
fi

# A rollback (SB-1805): the digests that ran before the last upgrade, which this cell
# admitted and attested then, and only one step back.
if [ "$MODE" = rollback ]; then
  [ -f "$DIR/previous.env" ] || die "no previous release to roll back to (only one step back is kept)"
  declare -A IMG REF REL
  while IFS='=' read -r k v; do
    case "$k" in SB_*_IMAGE) svc=${k#SB_}; svc=${svc%_IMAGE}; svc=${svc,,}; IMG[$svc]=$v ;; REF_*) REF[${k#REF_}]=$v ;; REL_*) REL[${k#REL_}]=$v ;; esac
  done <"$DIR/previous.env"
  for svc in "${SERVICES[@]}"; do
    [ -n "${IMG[$svc]:-}" ] && [ -n "${REF[$svc]:-}" ] || die "the previous record has no $svc"
    key=$(printf '%s' "${REF[$svc]}" | sha256sum | cut -d' ' -f1)
    state "test -f /s/deploy/attested/$key" >/dev/null || die "$svc's previous image ${REF[$svc]} isn't attested by this cell"
    "$DOCKER" image inspect "${IMG[$svc]}" >/dev/null 2>&1 || die "this machine no longer holds $svc's previous image"
  done
  tmp=$(mktemp)
  { grep -vE '^SB_(API|INGEST|JOBS|UPDATER)_IMAGE=' "$ENV" || true
    for svc in "${SERVICES[@]}"; do printf 'SB_%s_IMAGE=%s\n' "${svc^^}" "${IMG[$svc]}"; done; } >"$tmp"
  mv "$tmp" "$ENV"
  compose up -d >/dev/null
  for svc in api ingest jobs; do
    got=$("$DOCKER" inspect -f '{{.Image}}' "sb-selfhost-$svc-1" | tr -d '\r')
    [ "$got" = "${IMG[$svc]}" ] || { echo "selfhost-update: $svc runs $got, not ${IMG[$svc]}" >&2; exit 1; }
  done
  rec=
  for svc in "${SERVICES[@]}"; do
    rec+="printf '{\"image\":\"%s\",\"release\":\"%s\"}' '${REF[$svc]}' '${REL[$svc]}' >/s/deploy/running/$svc.json; "
  done
  state "mkdir -p /s/deploy/running && $rec chown -R 65532:65532 /s/deploy/running && chmod -R u=rwX,go= /s/deploy/running" >/dev/null
  printf '%s\n' "${REL[api]}" >"$DIR/held"
  rm -f "$DIR/previous.env" # one step back, never two
  echo "selfhost-update: rolled back to ${REL[api]} and held; update.sh --resume lifts the hold"
  exit 0
fi
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

# What runs now, kept for one rollback (SB-1805) when this upgrade changes a digest.
prev=
for svc in "${SERVICES[@]}"; do
  cur=$(sed -nE "s/^SB_${svc^^}_IMAGE=//p" "$ENV" | tr -d '\r')
  run=$(state "cat /s/deploy/running/$svc.json 2>/dev/null || true")
  ref=$(printf '%s' "$run" | sed -nE 's/.*"image":"([^"]+)".*/\1/p'); rel=$(printf '%s' "$run" | sed -nE 's/.*"release":"([^"]+)".*/\1/p')
  [ -n "$cur" ] && [ -n "$ref" ] || { prev=; break; }
  prev+="SB_${svc^^}_IMAGE=$cur"$'\n'"REF_$svc=$ref"$'\n'"REL_$svc=$rel"$'\n'
done
changed=
for svc in "${SERVICES[@]}"; do [ "$(sed -nE "s/^SB_${svc^^}_IMAGE=//p" "$ENV" | tr -d '\r')" = "${IMG[$svc]}" ] || changed=1; done
if [ -n "$changed" ] && [ -n "$prev" ]; then printf '%s' "$prev" >"$DIR/previous.env"; fi

tmp=$(mktemp)
{
  grep -vE '^SB_(API|INGEST|JOBS|UPDATER)_IMAGE=' "$ENV" || true  # SB_MODEL_IMAGE is the model profile's own pin (SB-1710)
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
