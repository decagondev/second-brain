#!/usr/bin/env bash
# make selfhost-smoke (SB-1601, SB-1602): checks a running self-hosted cell from outside,
# as sb-selfhost(7) describes it. Run after `make selfhost`.
#   - every cell service is up, as the non-root user, with a read-only root filesystem
#   - the proxy serves /readyz and the web UI over TLS, and sign-in then an ask work
#   - the database network has no route out
#   - each vault is owned by the service user and closed to everyone else
#   - answers cite the example share, and never a file the asker can't read or one with no ACL (SB-1606)
#   - anchors land in the write-once store, whose lock refuses a delete (SB-1603)
#   - no container holds the Docker socket; each runs the image the updater admitted, and
#     a release signed with another key changes nothing (SB-1604)
#   - the bundled sign-in refuses a wrong password and signs the demo user in with the right
#     one; the access job leaves a statement (SB-1605)
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
compose() { DOCKER="$DOCKER" bash tools/selfhost/compose.sh "$@"; }
mkdir -p build/selfhost; tmpd=$(mktemp -d build/selfhost/.smoke.XXXXXX); trap 'rm -rf "$tmpd"' EXIT
win() { if [[ "$CURL" == *.exe ]] && command -v wslpath >/dev/null; then wslpath -w "$1"; else printf '%s' "$1"; fi; }
jarp=$(win "$tmpd/jar")
NUL=/dev/null; [[ "$CURL" == *.exe ]] && NUL=NUL  # curl.exe can't write to /dev/null
# signin PASSWORD [EMAIL]: the browser's part of the bundled sign-in (sb-idp(1)): the form, then the post.
signin() {
  local page="$tmpd/form.html" url
  url=$("$CURL" -ks "${R[@]}" -c "$jarp" -b "$jarp" -L -o "$(win "$page")" -w '%{url_effective}' https://cell.localhost:8443/login) || return 1
  local args=() n v
  for n in state nonce code_challenge at csrf; do
    v=$(sed -nE "s/.*name=\"$n\" value=\"([^\"]*)\".*/\1/p" "$page" | head -1 | sed 's/&#43;/+/g')
    args+=(--data-urlencode "$n=$v")
  done
  local out
  out=$("$CURL" -ks "${R[@]}" -c "$jarp" -b "$jarp" -o $NUL -w '%{http_code} %{redirect_url}' "${args[@]}" \
    --data-urlencode "email=${2:-ann@acme.example}" --data-urlencode "password=$1" "$url") || return 1
  # A sign-in goes back to the cell's callback, which sets the session cookie.
  if [ "${out%% *}" = 303 ]; then "$CURL" -ks "${R[@]}" -c "$jarp" -b "$jarp" -L -o $NUL "${out#* }"; fi
  printf '%s' "${out%% *}"
}
if [ -f build/selfhost/demo-password ]; then
  # A wrong password for an address that isn't a user, so reruns never lock ann out.
  check "the bundled sign-in refuses a wrong password" "[ \"\$(signin 'not-the-password-at-all' nobody@acme.example)\" = 401 ]"
  code=$(signin "$(cat build/selfhost/demo-password)")
  check "the bundled sign-in takes the right one" "[ '$code' = 303 ]"
else
  "$CURL" -ks "${R[@]}" -c "$jarp" -b "$jarp" -L -o $NUL https://cell.localhost:8443/login || true
fi
check "sign in, then ask" "\"$CURL\" -ksf ${R[*]} -b '$jarp' -X POST https://cell.localhost:8443/v1/ask -H 'Content-Type: application/json' -d '{\"question\":\"hello\"}' | grep -q '\"trace\"'"
ask() { "$CURL" -ksf "${R[@]}" -b "$jarp" -X POST https://cell.localhost:8443/v1/ask -H 'Content-Type: application/json' -d "{\"question\":\"$1\"}"; }
ok=
for _ in $(seq 30); do ask "how long do customers have to return a pallet?" | grep -q 'docs:operations/returns-policy.md@' && { ok=1; break; }; sleep 2; done
check "an answer cites the example share (SB-1606)" "[ -n '$ok' ]"
check "a file ann can't read is never cited" "! ask 'what are the salary bands?' | grep -q 'docs:people/'"
check "a file with no ACL file is never cited" "! ask 'what is the example share?' | grep -q 'docs:README.md'"
# S16.2: stewards' notes are the one folder of the share api may write; nobody else writes it.
for svc in api ingest jobs; do
  want=false; [ "$svc" = api ] && want=true
  check "$svc mounts the share read-only and notes/ writable=$want (SB-1705)" \
    "[ \"\$(\"$DOCKER\" inspect -f '{{range .Mounts}}{{.Destination}}={{.RW}} {{end}}' sb-selfhost-$svc-1 | tr ' ' '\n' | grep -E '^/srv/share(/notes)?=' | sort | tr '\n' ' ' | tr -d '\r')\" = '/srv/share=false /srv/share/notes=$want ' ]"
done
# S16.5: with the model profile, both model servers sit on the data network alone, publish
# nothing, run as the service user on a read-only root, and the cell answers through them.
if [ -f build/selfhost/model.json ]; then
  for svc in model model-embed; do
    c=sb-selfhost-$svc-1
    check "$svc is on the data network only, with no published port (SB-1710)" \
      "[ \"\$(\"$DOCKER\" inspect -f '{{range \$k, \$v := .NetworkSettings.Networks}}{{\$k}} {{end}}|{{len .NetworkSettings.Ports}}' $c | tr -d '\r')\" = 'sb-selfhost_data |0' ]"
    check "$svc runs as 65532 on a read-only root (SB-1710)" "[ \"\$(\"$DOCKER\" inspect -f '{{.Config.User}} {{.HostConfig.ReadonlyRootfs}}' $c | tr -d '\r')\" = '65532:65532 true' ]"
  done
  check "an answer comes from the model on the box (SB-1710)" "ask 'how long do customers have to return a pallet?' | grep -q 'docs:operations/returns-policy.md@'"
  # The answer latency to expect on this machine (sb-selfhost(7), Hardware and latency).
  ts=()
  for q in "how long do customers have to return a pallet?" "who approves a refund?" "what is in the returns policy?" "how are returns inspected?" "what happens to a damaged pallet?"; do
    ts+=("$("$CURL" -ksf -o $NUL -w '%{time_total}' "${R[@]}" -b "$jarp" -X POST https://cell.localhost:8443/v1/ask -H 'Content-Type: application/json' -d "{\"question\":\"$q\"}" || echo fail)")
  done
  echo "info  answer latency with the model on the box, 5 asks (s): ${ts[*]}"
fi
if [ -f build/selfhost/demo-password ]; then
  # S15.4: the demo's operator sees the cell's health and the unhelpful-answer queue.
  check "the console's health answers (SB-1607)" "\"$CURL\" -ksf ${R[*]} -b '$jarp' https://cell.localhost:8443/v1/admin/health | grep -q '\"connectors\":\\[{\"kind\":\"docs\"'"
  check "the unhelpful-answer queue answers (SB-1609)" "\"$CURL\" -ksf ${R[*]} -b '$jarp' 'https://cell.localhost:8443/v1/admin/feedback?rating=down' | grep -q '\"items\"'"
fi
check "asking without signing in is refused" "! \"$CURL\" -ksf ${R[*]} -X POST https://cell.localhost:8443/v1/ask -d '{}'"
check "the database network has no route out" "! \"$DOCKER\" run --rm --network sb-selfhost_data $HELPER wget -q -T 5 -O /dev/null http://example.com"
for svc in api ingest jobs; do
  check "the $svc vault is 65532's alone" "[ -z \"\$(\"$DOCKER\" run --rm -v sb-selfhost_vault-$svc:/v:ro $HELPER find /v ! -user 65532 -o -perm /077 | tr -d '\r')\" ]"
done
check "ingest holds no sign-in secret" "! \"$DOCKER\" run --rm -v sb-selfhost_vault-ingest:/v:ro $HELPER test -e /v/vault/tenants/acme-logistics/oidc"

# S15.2: the write-once store and upgrades.
ENVF=build/selfhost/compose.env
check "the anchor job writes to the write-once store" "compose run --rm jobs --config /etc/sb/cell.json run anchor"
check "anchors are in the store" "[ -n \"\$(\"$DOCKER\" run --rm -v sb-selfhost_worm-data:/d:ro $HELPER find /d/objects/sb-worm/anchors -type f | tr -d '\r')\" ]"
check "the store refuses to delete a locked object" "compose --profile setup run --rm wormsetup"
# A night's copies are write-once, so a second run the same day is refused: run it, then
# look for tonight's copy in all three places.
compose run --rm jobs --config /etc/sb/cell.json run backup >/dev/null 2>&1 || true
night=acme-logistics/$(date -u +%F)
check "tonight's backup is on the box" "\"$DOCKER\" run --rm -v sb-selfhost_state:/s:ro $HELPER test -s /s/backups/$night/audit.ndjson"
check "and in the write-once store" "[ -n \"\$(\"$DOCKER\" run --rm -v sb-selfhost_worm-data:/d:ro $HELPER find /d/objects/sb-worm/$night -type f | tr -d '\r')\" ]"
check "and off the box (SB-1605)" "\"$DOCKER\" run --rm -v \"$(win "$(pwd)/build/selfhost/.offsite"):/o:ro\" $HELPER test -s /o/$night/audit.ndjson"
check "the access job leaves a statement (LAW-17 on one machine)" "compose run --rm jobs --config /etc/sb/cell.json run access && \"$DOCKER\" run --rm -v sb-selfhost_state:/s:ro $HELPER sh -c 'ls /s/access/statement-*.json'"
# sb tenant verify against the running cell, from a container on its edge network, trusting
# the proxy's CA and holding only the control plane's identity (sb-tenant(1), SB-1605).
if command -v go >/dev/null; then
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$tmpd/sb" ./cmd/sb
  vfy() {
    "$DOCKER" run --rm --network sb-selfhost_edge -e SB_TOKEN="$(cat build/selfhost/vault/api/vault/tenants/acme-logistics/control-plane)"       -e SSL_CERT_FILE=/w/ca/caddy-root.crt -v "$(win "$(pwd)/build/selfhost"):/w:ro" $HELPER       "/w/$(basename "$tmpd")/sb" tenant verify /w/tenant.yaml --cell https://cell.localhost:8443 --sample 5
  }
  check "sb tenant verify --cell is green" "vfy | grep -q 'verify: [0-9]* passed, 0 failed'"
fi
check "no container holds the Docker socket" "! \"$DOCKER\" inspect -f '{{range .Mounts}}{{.Source}} {{end}}' \$(\"$DOCKER\" ps -q --filter label=com.docker.compose.project=sb-selfhost | tr -d '\r') | grep -q docker.sock"
if grep -q '^SB_API_IMAGE=' "$ENVF" 2>/dev/null; then
  for svc in api ingest jobs; do
    want=$(sed -nE "s/^SB_${svc^^}_IMAGE=//p" "$ENVF")
    check "$svc runs the image the updater admitted" "[ \"\$(\"$DOCKER\" inspect -f '{{.Image}}' sb-selfhost-$svc-1 | tr -d '\r')\" = '$want' ]"
  done
fi
# The demo only (its release key is here): a release signed with a key this cell doesn't
# trust is refused, nothing changes, and the channel is put back.
if [ -f build/selfhost/release-key ] && command -v go >/dev/null; then
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
