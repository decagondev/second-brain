#!/usr/bin/env bash
# install.sh --support (sb-selfhost(7), Support bundle; SB-1808): one tar to send to whoever
# helps, with no content and no secret in it. It holds the versions and image digests, the
# layout's settings (compose.env and cell.json: names, paths and digests; secrets there are
# vault references), the doctor's verdicts and the facts behind them, `ps`, and the last
# 1,000 log lines of each service. Nothing under /etc/second-brain or build/selfhost/vault is
# copied in. Before the tar is written, every file in it is scanned by `sb secrets scan`
# against every value stored in the vaults and the box's own secret files, and for the
# shapes of secrets; any finding stops it and nothing is written.
#   support.sh [OUT]     default build/selfhost/support-TIME.tar.gz; DOCKER names Docker
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
DIR=build/selfhost
ENV=$DIR/compose.env
die() { echo "support: $*" >&2; exit 1; }
compose() { DOCKER="$DOCKER" bash tools/selfhost/compose.sh "$@"; }
[ -f "$ENV" ] || die "no self-hosted cell here"
OUT=${1:-$DIR/support-$(date -u +%Y%m%dT%H%M%SZ).tar.gz}
stage=$(mktemp -d "$DIR/.support.XXXXXX")
chmod 700 "$stage"
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/settings" "$stage/logs"

{
  echo "made: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "checkout: $(git rev-parse HEAD 2>/dev/null || echo "an install bundle")"
  echo "sb: $(bash tools/selfhost/sb.sh version 2>/dev/null | head -1)"
  echo "docker: $("$DOCKER" version --format '{{.Server.Version}}' 2>/dev/null || echo unknown)"
  echo "compose: $("$DOCKER" compose version --short 2>/dev/null || echo unknown)"
  echo "kernel: $(uname -sr)"
  echo "images running:"
  for c in $("$DOCKER" ps -q --filter label=com.docker.compose.project=sb-selfhost 2>/dev/null); do
    "$DOCKER" inspect --format '  {{.Name}} {{.Image}}' "$c" 2>/dev/null || true
  done
} >"$stage/versions.txt"
cp "$ENV" "$stage/settings/compose.env"
[ -f "$DIR/cell.json" ] && cp "$DIR/cell.json" "$stage/settings/cell.json"
[ -f "$DIR/drill.json" ] && cp "$DIR/drill.json" "$stage/settings/drill.json"
[ -f "$DIR/held" ] && cp "$DIR/held" "$stage/settings/held"
bash tools/selfhost/facts.sh >"$stage/facts.json" 2>/dev/null || true
bash tools/selfhost/sb.sh selfhost doctor --facts "$stage/facts.json" >"$stage/doctor.txt" 2>&1 || true
compose ps -a >"$stage/ps.txt" 2>&1 || true
for svc in $(compose config --services 2>/dev/null); do
  compose logs --no-color --tail 1000 "$svc" >"$stage/logs/$svc.log" 2>&1 || true
done

# Every value the box stores, so the scan knows them: the vaults and the files beside them.
known=(--known "$DIR/vault")
for f in db-password worm-secret demo-password release-key; do
  [ -e "$DIR/$f" ] && known+=(--known "$DIR/$f")
done
bash tools/selfhost/sb.sh secrets scan "${known[@]}" "$stage" >&2 ||
  die "the bundle would carry a secret (the lines above, by file and line); nothing was written"
tar -C "$stage" -czf "$OUT.tmp" . && mv "$OUT.tmp" "$OUT"
chmod 600 "$OUT"
echo "support: $OUT (no content, no secret; scanned)"
