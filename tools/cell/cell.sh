#!/usr/bin/env bash
# make cell (SB-1508): a whole cell on this machine. It builds the services, makes a local
# Postgres database, writes an example tenant's configuration, vault and manifest under
# SB_CELL_DIR (default ~/.sb-cell, on a filesystem with Unix permissions: the vault refuses
# a secret anyone else can read, and Windows drives mounted in WSL show every file as
# 0777), and starts the development IdP, sb-api, sb-ingest and sb-jobs against it.
# `make cell-stop` stops them. Nothing here is for a real tenant: the IdP has no passwords.
#
#   SB_CELL_DSN   the database (default postgres:///sb_cell?host=/var/run/postgresql)
#   SB_CELL_PORT  the api's port (default 8080); the IdP takes the next one
set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT=$(pwd)
DIR=${SB_CELL_DIR:-$HOME/.sb-cell}
DSN=${SB_CELL_DSN:-postgres:///sb_cell?host=/var/run/postgresql}
PORT=${SB_CELL_PORT:-8080}
IDP_PORT=$((PORT + 1))
TENANT=acme-logistics

if [ "${1:-}" = stop ]; then
  for p in "$DIR"/run/*.pid; do
    [ -f "$p" ] || continue
    kill "$(cat "$p")" 2>/dev/null || true
    rm -f "$p"
  done
  echo "cell: stopped"
  exit 0
fi

mkdir -p "$DIR/run" "$DIR/state" "$DIR/vault/vault/tenants/$TENANT/connectors" "$DIR/vault/vault/tenants/$TENANT/audit" "$DIR/policy/platform" "$DIR/logs"
chmod 700 "$DIR/vault"

db=$(printf '%s' "$DSN" | sed -E 's#^postgres(ql)?://[^/]*/([^?]+).*#\2#')
host=$(printf '%s' "$DSN" | sed -nE 's#.*host=([^&]+).*#\1#p')
if ! psql "$DSN" -Atc 'select 1' >/dev/null 2>&1; then
  createdb ${host:+-h "$host"} "$db"
fi
psql "$DSN" -qc 'CREATE EXTENSION IF NOT EXISTS vector' >/dev/null

secret() { # path value: a vault secret readable by its owner only
  local f="$DIR/vault/vault/tenants/$TENANT/$1"
  [ -f "$f" ] || { printf '%s' "$2" >"$f"; chmod 600 "$f"; }
}
rand() { head -c 32 /dev/urandom | base64 | tr -d '\n'; }
secret oidc "$(rand)"
secret scim "$(rand)"
secret control-plane "$(rand)"
secret audit/anchor-seed "$(head -c 32 /dev/urandom | base64 | tr -d '\n')"
secret attest-seed "$(head -c 32 /dev/urandom | base64 | tr -d '\n')"

cp policy/platform/*.cedar "$DIR/policy/platform/"
cp examples/policies/acme-logistics.cedar "$DIR/policy/"
[ -f "$DIR/tenant.yaml" ] || cat >"$DIR/tenant.yaml" <<YAML
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
  "manifest": "$DIR/tenant.yaml",
  "state": "$DIR/state",
  "vault": "$DIR/vault",
  "policies": "$DIR/policy",
  "listen": { "api": "127.0.0.1:$PORT" },
  "base_url": "http://localhost:$PORT",
  "oidc": { "client_id": "sb-$TENANT", "issuer": "http://127.0.0.1:$IDP_PORT" },
  "model": { "provider": "extractive" },
  "database": "$DSN",
  "durable": true
}
JSON

make -s build >/dev/null
go build -o bin/devidp ./tools/devidp

start() { # name command...
  local name=$1; shift
  nohup "$@" >"$DIR/logs/$name.log" 2>&1 &
  echo $! >"$DIR/run/$name.pid"
}
start devidp bin/devidp --listen "127.0.0.1:$IDP_PORT" --tenant "$TENANT" --client "sb-$TENANT" \
  --secret-file "$DIR/vault/vault/tenants/$TENANT/oidc"
for _ in $(seq 50); do curl -fs "http://127.0.0.1:$IDP_PORT/.well-known/openid-configuration" >/dev/null && break; sleep 0.1; done
start api bin/sb-api --config "$DIR/cell.json"
for _ in $(seq 100); do curl -fs "http://127.0.0.1:$PORT/readyz" >/dev/null && break; sleep 0.1; done
if ! curl -fs "http://127.0.0.1:$PORT/readyz" >/dev/null; then
  echo "cell: the api didn't become ready; see $DIR/logs/api.log" >&2
  "$0" stop >/dev/null
  exit 1
fi
start ingest bin/sb-ingest --config "$DIR/cell.json"
start jobs bin/sb-jobs --config "$DIR/cell.json"
echo "cell: $TENANT is running at http://localhost:$PORT (IdP http://127.0.0.1:$IDP_PORT)"
echo "cell: configuration $DIR/cell.json, logs $DIR/logs; SCIM token in $DIR/vault/vault/tenants/$TENANT/scim"
echo "cell: make cell-stop stops it"
