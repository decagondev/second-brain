#!/usr/bin/env bash
# install.sh --recovery-kit (sb-selfhost(7), The recovery kit; SB-1902): take the recovery
# key off the box, into five shares on paper, any three of which rebuild it.
#
#   kit.sh [--new-key] [--custodian N=NAME]...
#
# It archives the box's keys (the vaults, the database and write-once store secrets, the
# recovery key pair, the demo's release key; not the bundled IdP's volume, whose people are
# added again) and seals the archive with a fresh kit key,
# which `sb recovery split` splits into five share pages. The sealed archive and the kit's
# record (kit, tenant, date, who holds which share; never a share) are kept on the box and
# copied off it. Then the recovery key's private half is deleted from the box. Print each
# share page, give it to its custodian, and delete the pages: the doctor warns until then.
# --new-key starts a new recovery key first, for when the old kit is lost: copies sealed
# before stay under the old key.
set -euo pipefail
cd "$(dirname "$0")/../.."
DIR=build/selfhost
ENV=$DIR/compose.env
die() { echo "kit: $*" >&2; exit 1; }
[ -f "$ENV" ] || die "no self-hosted cell here"
NEW= CUST=()
while [ $# -gt 0 ]; do
  case "$1" in
  --new-key) NEW=1 ;;
  --custodian) CUST+=(--custodian "${2:?--custodian needs N=NAME}"); shift ;;
  *) die "usage: install.sh --recovery-kit [--new-key] [--custodian N=NAME]..." ;;
  esac
  shift
done
[ -f "$DIR/recovery.pub" ] || die "off-box sealing is off (SB_SEAL=off): there is no recovery key to put in a kit"
if [ -n "$NEW" ]; then
  [ ! -f "$DIR/recovery-key.private" ] || die "the recovery key's private half is still here: make the kit from it, without --new-key"
  id=$(date -u +%Y%m%dT%H%M%SZ)
  mv "$DIR/recovery.pub" "$DIR/recovery-$id.pub.old"
  bash tools/selfhost/sb.sh backup keygen --public "$DIR/recovery.pub" --private "$DIR/recovery-key.private" >/dev/null
  echo "kit: a new recovery key; copies sealed before stay under the old one (its kit)" >&2
  echo "kit: run install.sh CONFIG again so the nightly backup seals to the new key" >&2
fi
[ -f "$DIR/recovery-key.private" ] || die "the recovery key's private half isn't on this box (a kit was made: $(ls "$DIR"/kit/kit_*.json 2>/dev/null | head -1)); --new-key starts a new one"
TENANT=$(sed -nE 's/^SB_TENANT=//p' "$ENV" | tr -d '\r')
mkdir -p "$DIR/kit" && chmod 700 "$DIR/kit"
arch=$(mktemp "$DIR/kit/.keys.XXXXXX")
trap 'rm -f "$arch"' EXIT
files=(vault recovery.pub recovery-key.private)
for f in db-password worm-secret release-key; do [ -e "$DIR/$f" ] && files+=("$f"); done
tar -C "$DIR" -cf "$arch" "${files[@]}"
out=$(bash tools/selfhost/sb.sh recovery split --tenant "$TENANT" --out "$DIR/kit" "${CUST[@]}" "$arch")
kit=$(printf '%s\n' "$out" | sed -nE 's/^(kit_[0-9a-f]+):.*/\1/p' | head -1)
[ -n "$kit" ] || die "the kit wasn't made"
off=$(sed -nE 's/^SB_OFFSITE=//p' "$ENV" | tr -d '\r')
# shellcheck source=tools/selfhost/offsite.sh
. tools/selfhost/offsite.sh
if [ -n "$(off_url)" ]; then # the company's bucket (SB-2303): the sealed archive and its record
  off_put "$DIR/kit/$kit.sealed" "kit/$kit.sealed" && off_put "$DIR/kit/$kit.json" "kit/$kit.json" ||
    die "the kit couldn't be copied to $(off_url); the recovery key's private half stays until it is"
  off=$(off_url)
elif [ -n "$off" ] && [ -d "$off" ]; then
  mkdir -p "$off/kit"
  cp "$DIR/kit/$kit.sealed" "$DIR/kit/$kit.json" "$off/kit/"
fi
# The private half leaves the box: from now on only three shares rebuild it.
if command -v shred >/dev/null; then shred -u "$DIR/recovery-key.private"; else rm -f "$DIR/recovery-key.private"; fi
echo "kit: $kit made for $TENANT; the sealed keys are in $DIR/kit and ${off:-SB_OFFSITE}/kit"
echo "kit: print each page and give it to its custodian, then delete the pages from this machine:"
ls -1 "$DIR"/kit/"$kit"-share-*.txt | sed 's/^/  /'
echo "kit: rm $DIR/kit/$kit-share-*.txt"
