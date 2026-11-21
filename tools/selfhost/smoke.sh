#!/usr/bin/env bash
# make selfhost-smoke (SB-1601, SB-1602): checks a running self-hosted cell from outside,
# as sb-selfhost(7) describes it. Run after `make selfhost`.
#   - every cell service is up, as the non-root user, with a read-only root filesystem
#   - the proxy serves /readyz and the web UI over TLS, and sign-in then an ask work
#   - the database network has no route out
#   - each vault is owned by the service user and closed to everyone else
#   - anchors land in the write-once store, whose lock refuses a delete (SB-1603)
#   - no container holds the Docker socket; each runs the image the updater admitted, and
#     a release signed with another key changes nothing (SB-1604)
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f
R=(--resolve cell.localhost:8443:127.0.0.1 --resolve idp.localhost:8081:127.0.0.1)
fail=0
check() { if eval "$2" >/dev/null 2>&1; then echo "ok    $1"; else echo "FAIL  $1"; fail=1; fi; }

for svc in api ingest jobs; do
  c=sb-selfhost-$svc-1
  check "$svc is running" "[ \"\$(\"$DOCKER\" inspect -f '{{.State.Running}}' $c | tr -d '\r')\" = true ]"
  check "$svc runs as 65532, read-only root" "[ \"\$(\"$DOCKER\" inspect -f '{{.Config.User}} {{.HostConfig.ReadonlyRootfs}}' $c | tr -d '\r')\" = '65532:65532 true' ]"
done
check "readyz through the proxy" "\"$CURL\" -ksf ${R[*]} https://cell.localhost:8443/readyz"
check "the web UI through the proxy" "\"$CURL\" -ksf ${R[*]} https://cell.localhost:8443/ | grep -q '<div id=\"root\">'"
jar=$(mktemp -d)/jar
if [[ "$CURL" == *.exe ]] && command -v wslpath >/dev/null; then jarp=$(wslpath -w "$jar"); else jarp=$jar; fi
"$CURL" -ks "${R[@]}" -c "$jarp" -b "$jarp" -L -o /dev/null https://cell.localhost:8443/login || true
check "sign in, then ask" "\"$CURL\" -ksf ${R[*]} -b '$jarp' -X POST https://cell.localhost:8443/v1/ask -H 'Content-Type: application/json' -d '{\"question\":\"hello\"}' | grep -q '\"trace\"'"
check "asking without signing in is refused" "! \"$CURL\" -ksf ${R[*]} -X POST https://cell.localhost:8443/v1/ask -d '{}'"
check "the database network has no route out" "! \"$DOCKER\" run --rm --network sb-selfhost_data $HELPER wget -q -T 5 -O /dev/null http://example.com"
for svc in api ingest jobs; do
  check "the $svc vault is 65532's alone" "[ -z \"\$(\"$DOCKER\" run --rm -v sb-selfhost_vault-$svc:/v:ro $HELPER find /v ! -user 65532 -o -perm /077 | tr -d '\r')\" ]"
done
check "ingest holds no sign-in secret" "! \"$DOCKER\" run --rm -v sb-selfhost_vault-ingest:/v:ro $HELPER test -e /v/vault/tenants/acme-logistics/oidc"

# S15.2: the write-once store and upgrades.
ENVF=build/selfhost/images.env
compose() { if [ -f "$ENVF" ]; then "$DOCKER" compose --env-file "$ENVF" -f deploy/selfhost/compose.yaml "$@"; else "$DOCKER" compose -f deploy/selfhost/compose.yaml "$@"; fi; }
check "the anchor job writes to the write-once store" "compose run --rm jobs --config /etc/sb/cell.json run anchor"
check "anchors are in the store" "[ -n \"\$(\"$DOCKER\" run --rm -v sb-selfhost_worm-data:/d:ro $HELPER find /d/objects/sb-worm/anchors -type f | tr -d '
')\" ]"
check "the store refuses to delete a locked object" "compose --profile setup run --rm wormsetup"
check "no container holds the Docker socket" "! \"$DOCKER\" inspect -f '{{range .Mounts}}{{.Source}} {{end}}' \$(\"$DOCKER\" ps -q --filter label=com.docker.compose.project=sb-selfhost | tr -d '
') | grep -q docker.sock"
if [ -f "$ENVF" ]; then
  for svc in api ingest jobs; do
    want=$(sed -nE "s/^SB_${svc^^}_IMAGE=//p" "$ENVF")
    check "$svc runs the image the updater admitted" "[ \"\$(\"$DOCKER\" inspect -f '{{.Image}}' sb-selfhost-$svc-1 | tr -d '
')\" = '$want' ]"
  done
  # A release signed with a key this cell doesn't trust: the updater refuses it, nothing
  # changes, and the channel is put back.
  rogue=$(mktemp -d)
  cp "$ENVF" "$rogue/before"
  d() { sed -nE "s/^$1 (sha256:[0-9a-f]{64})\$/\1/p" build/images/digests.txt | tr -d '\r'; }
  GOTOOLCHAIN=local go run ./tools/release/bundle --image api="$(d api)" --image ingest="$(d ingest)" --image jobs="$(d jobs)"     --image updater="$(d updater)" --created 2099-01-01 --out "$rogue/bundle.json" >/dev/null
  mkdir -p "$rogue/releases"
  if GOTOOLCHAIN=local go run ./tools/selfhost/devrelease --bundle "$rogue/bundle.json" --store "$rogue/releases" --key-file "$rogue/key" --gate-results build/selfhost/gate-results.txt >/dev/null 2>&1; then
    rp=$rogue/releases
    if [[ "$DOCKER" == *.exe ]] && command -v wslpath >/dev/null; then rp=$(wslpath -w "$rp"); fi
    "$DOCKER" run --rm -u 0 -v sb-selfhost_releases:/dst -v "$rp:/src:ro" $HELPER sh -c 'cp -a /src/. /dst/ && chown -R 65532:65532 /dst' >/dev/null
    check "a release signed with another key is refused" "! DOCKER=\"$DOCKER\" bash tools/selfhost/update.sh"
    check "and nothing changed" "cmp -s '$rogue/before' '$ENVF'"
    GOTOOLCHAIN=local go run ./tools/selfhost/devrelease --bundle build/selfhost/bundle.json --store build/selfhost/releases --key-file build/selfhost/release-key --gate-results build/selfhost/gate-results.txt >/dev/null
    rp=build/selfhost/releases
    if [[ "$DOCKER" == *.exe ]] && command -v wslpath >/dev/null; then rp=$(wslpath -w "$rp"); fi
    "$DOCKER" run --rm -u 0 -v sb-selfhost_releases:/dst -v "$rp:/src:ro" $HELPER sh -c 'cp -a /src/. /dst/ && chown -R 65532:65532 /dst' >/dev/null
  else
    echo "FAIL  the rogue release couldn't be made"; fail=1
  fi
  rm -rf "$rogue"
fi
exit $fail
