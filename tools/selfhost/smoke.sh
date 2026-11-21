#!/usr/bin/env bash
# make selfhost-smoke (SB-1601, SB-1602): checks a running self-hosted cell from outside,
# as sb-selfhost(7) describes it. Run after `make selfhost`.
#   - every cell service is up, as the non-root user, with a read-only root filesystem
#   - the proxy serves /readyz and the web UI over TLS, and sign-in then an ask work
#   - the database network has no route out
#   - each vault is owned by the service user and closed to everyone else
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
exit $fail
