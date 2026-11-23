#!/usr/bin/env bash
# install.sh --retire (sb-selfhost(7), Retiring a box; SB-1907): destroy a self-hosted box
# for good, and leave a signed statement that says what was destroyed.
#
#   retire.sh [CONFIG] --yes-destroy
#
# Before anything is destroyed it anchors the audit log, exports it from the cell and
# checks the export under the cell's own key, and puts a copy off the box, sealed to the
# recovery key like every other off-box copy. Then it stops the cell and removes its
# systemd units, its containers and every volume (the database, the write-once store, the
# vaults, the sign-in), overwrites then deletes every file under build/selfhost (the
# keys, the passwords, the local dumps), and checks that none is left. Last it writes the
# statement, signed with an Ed25519 key made for it whose private half is never written:
# the public key is printed, to be kept with the statement. The sealed copies off the box
# stay; they open only with the recovery kit (any 3 of 5 shares). Without --yes-destroy it
# says what it would destroy and stops.
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
DIR=build/selfhost
ENV=$DIR/compose.env
die() { echo "retire: $*" >&2; exit 1; }
YES=
for a in "$@"; do
  case "$a" in
  --yes-destroy) YES=1 ;;
  -*) die "usage: install.sh --retire [CONFIG] --yes-destroy" ;;
  *) [ -f "$a" ] || die "no configuration at $a" ;; # accepted, as install.sh takes it; compose.env says the rest
  esac
done
[ -f "$ENV" ] || die "no self-hosted cell here (no $ENV)"
val() { sed -nE "s/^$1=//p" "$ENV" | tr -d '\r' | tail -1; }
compose() { DOCKER="$DOCKER" bash tools/selfhost/compose.sh "$@"; }
sb() { bash tools/selfhost/sb.sh "$@"; }
TENANT=$(val SB_TENANT) HOST=$(val SB_HOST) PORT=$(val SB_PORT) OFF=$(val SB_OFFSITE) DB=$(val SB_DATABASE)
if [[ "$DOCKER" == *.exe ]] && command -v wslpath >/dev/null; then OFF=$(wslpath -u "$OFF"); fi
# shellcheck source=tools/selfhost/offsite.sh
. tools/selfhost/offsite.sh
S3=$(off_url) # the company's bucket (SB-2303): what is kept goes there, sealed
[ -z "$S3" ] || OFF=$S3
SEED=$DIR/vault/api/vault/tenants/$TENANT/audit/anchor-seed
[ -f "$SEED" ] || die "the cell's audit key isn't here ($SEED): nothing to retire, or retired already"
# The compose project's volumes, and any other volume by its name (one made outside compose).
volumes() {
  { "$DOCKER" volume ls -q --filter label=com.docker.compose.project=sb-selfhost; "$DOCKER" volume ls -q --filter name=^sb-selfhost_; } |
    tr -d '\r' | sort -u
}
files=$(find "$DIR" -type f | wc -l | tr -d ' ')

if [ -z "$YES" ]; then
  echo "retire: this would destroy $TENANT's box for good:"
  echo "  the cell's containers and systemd units"
  echo "  its volumes: $(volumes | tr '\n' ' ')"
  echo "  $files files under $DIR, each overwritten then deleted (every key and password on the box)"
  echo "  and keep only the sealed copies in $OFF, which open with the recovery kit"
  die "refusing without --yes-destroy"
fi
command -v shred >/dev/null || die "shred (coreutils) is needed to overwrite the keys"
TS=$(date -u +%Y%m%dT%H%M%SZ)
OUT=build/retired/$TENANT-$TS
mkdir -p "$OUT"
[ -n "$S3" ] || mkdir -p "$OFF/retired"
if [ -n "$S3" ]; then
  [ -f "$DIR/recovery.pub" ] || die "SB_OFFSITE is a bucket but there is no recovery key to seal the export to"
  # The key pair outlives build/selfhost only in this shell, for the statement's copy.
  OFF_KEYS_JSON=$(cat "$(off_keys)") || die "the bucket's key pair isn't here ($(off_keys))"
fi
chmod 700 build/retired "$OUT"

# 1. The audit log, anchored and exported while the cell runs, checked before anything goes.
compose run --rm jobs --config /etc/sb/cell.json run anchor >/dev/null 2>&1 || echo "retire: the anchor job didn't run; the export signs its own last anchor" >&2
ca=; [ -f "$DIR/ca/caddy-root.crt" ] && ca=$DIR/ca/caddy-root.crt; [ -f "$DIR/ca/company-ca.crt" ] && ca=$DIR/ca/company-ca.crt
SSL_CERT_FILE=${ca:-${SSL_CERT_FILE:-}} SB_TOKEN=$(cat "$DIR/vault/api/vault/tenants/$TENANT/control-plane") \
  sb -q --tenant "$TENANT" audit export --cell "https://$HOST$([ "$PORT" = 443 ] || echo ":$PORT")" --since 1970-01-01T00:00:00Z --out "$OUT/audit.ndjson" ||
  die "the audit log couldn't be exported, so nothing was destroyed"
CELLKEY=$(sb selfhost retire-statement --check --tenant "$TENANT" --export "$OUT/audit.ndjson" --anchor-seed "$SEED") ||
  die "the export doesn't verify under the cell's key, so nothing was destroyed"
# Audit is kept for its seven years off the box, sealed like every other off-box copy.
if [ -n "$S3" ]; then
  sb -q backup seal --to "$DIR/recovery.pub" --out "$OUT/audit.ndjson.sealed" "$OUT/audit.ndjson"
  off_put "$OUT/audit.ndjson.sealed" "retired/audit-$TENANT-$TS.ndjson.sealed" || die "the sealed export couldn't be copied to $S3, so nothing was destroyed"
  rm -f "$OUT/audit.ndjson.sealed"
  kept_audit="the audit log's export, sealed to the recovery key, at $S3/retired/audit-$TENANT-$TS.ndjson.sealed"
elif [ -f "$DIR/recovery.pub" ]; then
  sb -q backup seal --to "$DIR/recovery.pub" --out "$OFF/retired/audit-$TENANT-$TS.ndjson.sealed" "$OUT/audit.ndjson"
  kept_audit="the audit log's export, sealed to the recovery key, at retired/audit-$TENANT-$TS.ndjson.sealed"
else
  cp "$OUT/audit.ndjson" "$OFF/retired/audit-$TENANT-$TS.ndjson"
  kept_audit="the audit log's export (ids and hashes, no content; SB_SEAL=off) at retired/audit-$TENANT-$TS.ndjson"
fi
echo "retire: audit exported and checked ($CELLKEY); destroying the box"

# 2. The cell: units, containers, volumes.
units=
if command -v systemctl >/dev/null && ls /etc/systemd/system/sb-selfhost*.service >/dev/null 2>&1; then
  for u in /etc/systemd/system/sb-selfhost*; do
    systemctl disable --now "$(basename "$u")" >/dev/null 2>&1 || true
    units="$units $(basename "$u")"
    rm -f "$u"
  done
  systemctl daemon-reload || true
fi
vols=$(volumes | tr '\n' ' ' | sed 's/ $//')
compose --profile db --profile model down -v --remove-orphans >/dev/null 2>&1 || true
for v in $(volumes); do "$DOCKER" volume rm -f "$v" >/dev/null; done
[ -z "$(volumes)" ] || die "volumes are left: $(volumes | tr '\n' ' ')"

# 3. Every file under build/selfhost: overwritten, then deleted. The demo's off-box
# directory lives under it (make selfhost), so it is moved out first, not destroyed.
case "$(realpath -m "$OFF")/" in
"$(realpath -m "$DIR")"/*)
  mv "$OFF" "$OUT/offsite" && OFF=$OUT/offsite
  echo "retire: SB_OFFSITE was under $DIR; it is kept at $OFF" >&2 ;;
esac
find "$DIR" -type f -print0 | xargs -0 -r shred -u -z -n 1
rm -rf "$DIR"
[ ! -e "$DIR" ] || die "$DIR is still there"

# 4. The statement.
destroyed=(--destroyed "the cell's containers${units:+ and systemd units:$units}"
  --destroyed "its volumes: ${vols:-none}"
  --destroyed "$files files under build/selfhost, each overwritten then deleted: the vaults, the audit and attestation keys, the database and write-once store secrets, the release key, TLS keys, the recovery key's public half, local database dumps")
kept=(--kept "sealed off-box copies under $OFF (database dumps, the backup job's copies, the recovery kit's sealed archive): they open only with the recovery kit, any 3 of 5 shares"
  --kept "$kept_audit")
if [ "$DB" = external ]; then kept+=(--kept "the company's own database, which this box used but didn't own: drop it there"); fi
sb selfhost retire-statement --tenant "$TENANT" --host "$HOST" --export "$OUT/audit.ndjson" --cell-key "$CELLKEY" \
  "${destroyed[@]}" "${kept[@]}" --out "$OUT/statement.json" | tee "$OUT/statement.txt"
if [ -n "$S3" ]; then
  off_put "$OUT/statement.json" "retired/statement-$TENANT-$TS.json" || echo "retire: the statement's copy didn't reach $S3; keep $OUT/statement.json" >&2
  unset OFF_KEYS_JSON
else
  cp "$OUT/statement.json" "$OFF/retired/statement-$TENANT-$TS.json"
fi
shred -u -z -n 1 "$OUT/audit.ndjson" # its sealed copy is off the box
key=$(sed -nE 's/^key: //p' "$OUT/statement.txt")
cat <<EOF
retire: $TENANT's box is retired. The statement is $OUT/statement.json (and a copy in $OFF/retired).
retire: its key, kept nowhere else, is
  $key
retire: print it with the statement. Anyone can check it: sb selfhost verify-retirement STATEMENT --key $key
EOF
