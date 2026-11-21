#!/usr/bin/env bash
# make selfhost (sb-selfhost(7); SB-1602): an example tenant's cell in containers on this
# machine's Docker. It builds the images, writes a configuration, per-service vaults and a
# manifest under build/selfhost (secrets made fresh, never committed), copies each vault
# into its own volume owned by the service user and readable by nobody else, brings the
# layout up, waits for /readyz through the proxy, and provisions a demo user over SCIM.
#
#   selfhost.sh [up|stop|clean]   DOCKER names the Docker command (Docker Desktop's
#                                 docker.exe on Windows); CURL the curl that reaches the
#                                 published ports (curl.exe when Docker runs on Windows)
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
TENANT=acme-logistics
DIR=build/selfhost
COMPOSE=("$DOCKER" compose -f deploy/selfhost/compose.yaml)
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f  # has a shell, for the volume step
CELL=https://cell.localhost:8443
IDP=http://idp.localhost:8081

case "${1:-up}" in
stop) "${COMPOSE[@]}" stop; exit 0 ;;
clean) "${COMPOSE[@]}" down -v; rm -rf "$DIR"; exit 0 ;;
up) ;;
*) echo "usage: selfhost.sh [up|stop|clean]" >&2; exit 2 ;;
esac

hostpath() { # a path the Docker command understands
  if command -v wslpath >/dev/null && [[ "$DOCKER" == *.exe ]]; then wslpath -w "$1"; else realpath "$1"; fi
}
rand() { head -c 32 /dev/urandom | base64 | tr -d '\n/+=' | head -c 40; }

[ "${SKIP_IMAGES:-}" = 1 ] || DOCKER="$DOCKER" bash tools/selfhost/images.sh >/dev/null

if [ ! -f "$DIR/cell.json" ]; then
  mkdir -p "$DIR/policy/platform"
  cp policy/platform/*.cedar "$DIR/policy/platform/"
  cp examples/policies/$TENANT.cedar "$DIR/policy/"
  dbpw=$(rand)
  printf '%s' "$dbpw" >"$DIR/db-password"
  oidc=$(rand) scim=$(rand) cp=$(rand)
  anchor=$(head -c 32 /dev/urandom | base64 | tr -d '\n') attest=$(head -c 32 /dev/urandom | base64 | tr -d '\n')
  dsn="postgres://sb:$dbpw@postgres:5432/sb?sslmode=disable"
  put() { mkdir -p "$(dirname "$DIR/vault/$1/vault/tenants/$TENANT/$2")"; printf '%s' "$3" >"$DIR/vault/$1/vault/tenants/$TENANT/$2"; }
  # Each service's vault holds only what that service reads (sb-selfhost(7), The layout).
  for f in "oidc $oidc" "scim $scim" "control-plane $cp" "audit/anchor-seed $anchor" "attest-seed $attest" "database $dsn"; do put api ${f%% *} "${f#* }"; done
  put ingest database "$dsn"
  for f in "audit/anchor-seed $anchor" "database $dsn"; do put jobs ${f%% *} "${f#* }"; done
  cat >"$DIR/tenant.yaml" <<YAML
tenant: $TENANT
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: gdrive }
agents: [ask, draft-reply]
retention_days: 365
budget_usd_month: 1500
YAML
  cat >"$DIR/cell.json" <<JSON
{
  "tenant": "$TENANT",
  "manifest": "/var/lib/sb/state/tenant.yaml",
  "state": "/var/lib/sb/state",
  "vault": "/var/lib/sb/vault",
  "policies": "/etc/sb/policy",
  "listen": { "api": "0.0.0.0:8080" },
  "base_url": "$CELL",
  "oidc": { "client_id": "sb-$TENANT", "issuer": "$IDP" },
  "web": "/sb/web",
  "model": { "provider": "extractive" },
  "database": "vault://tenants/$TENANT/database",
  "durable": true
}
JSON
fi

# Fill the volumes: each vault owned by the service user (65532) with no access for others.
"${COMPOSE[@]}" create --no-recreate >/dev/null 2>&1 || "${COMPOSE[@]}" create >/dev/null
fill() { # volume source-dir
  "$DOCKER" run --rm -u 0 -v "sb-selfhost_$1:/dst" -v "$(hostpath "$2"):/src:ro" "$HELPER" \
    sh -c 'cp -a /src/. /dst/ && chown -R 65532:65532 /dst && chmod -R u=rwX,go= /dst' >/dev/null
}
for svc in api ingest jobs; do fill "vault-$svc" "$DIR/vault/$svc"; done
"$DOCKER" run --rm -u 0 -v sb-selfhost_state:/dst -v "$(hostpath "$DIR"):/src:ro" "$HELPER" \
  sh -c '[ -f /dst/tenant.yaml ] || cp /src/tenant.yaml /dst/; chown -R 65532:65532 /dst; chmod 700 /dst' >/dev/null

"${COMPOSE[@]}" up -d >/dev/null
ok=
for _ in $(seq 90); do
  if "$CURL" -ksf --resolve cell.localhost:8443:127.0.0.1 "$CELL/readyz" >/dev/null 2>&1; then ok=1; break; fi
  sleep 1
done
if [ -z "$ok" ]; then
  echo "selfhost: the cell didn't become ready; '${COMPOSE[*]} logs api' says why" >&2
  exit 1
fi

# A demo user, provisioned the way a company's IdP would: over SCIM.
scim=$(cat "$DIR/vault/api/vault/tenants/$TENANT/scim")
body=$("$CURL" -ks --resolve cell.localhost:8443:127.0.0.1 -X POST "$CELL/scim/v2/Users" -H "Authorization: Bearer $scim" \
  -H "Content-Type: application/scim+json" -d '{"userName":"ann","emails":[{"value":"ann@acme.example"}]}')
id=$(printf '%s' "$body" | sed -nE 's/.*"id":"(usr_[0-9a-f]+)".*/\1/p')
if [ -n "$id" ]; then printf '%s' "$id" >"$DIR/demo-user"; else id=$(cat "$DIR/demo-user" 2>/dev/null || true); fi # ann exists from an earlier run
[ -n "$id" ] && "$CURL" -s -X POST "$IDP/dev/user?id=$id" >/dev/null
echo "selfhost: $TENANT is running at $CELL (demo IdP $IDP)"
echo "selfhost: sign in as ann ($id) at $CELL/login; make selfhost-stop stops it, make selfhost-clean deletes its data"
