#!/usr/bin/env bash
# make selfhost (sb-selfhost(7); SB-1602 to SB-1605): one tenant's cell in containers on
# this machine's Docker. With no configuration it is the demo: an example tenant, the
# example share, the bundled sign-in with a demo user, and a release signed by a key made
# for this machine. With SB_CONFIG=FILE (deploy/selfhost/install.env.example; install.sh
# passes it) it is a real install: the company's manifest, IdP, TLS, release store and
# off-box backup directory.
#
# It writes build/selfhost (the configuration, per-service vaults, the Caddyfile and
# compose.env; secrets made on this machine, once, and never committed), copies each vault
# into its own volume owned by the service user and readable by nobody else, brings the
# layout up, makes and proves the write-once bucket, and waits for /readyz.
#
#   selfhost.sh [up|stop|clean]   DOCKER names the Docker command (Docker Desktop's
#                                 docker.exe on Windows); CURL the curl that reaches the
#                                 published ports (curl.exe when Docker runs on Windows)
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
DIR=build/selfhost
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f  # has a shell, for the volume step
compose() { DOCKER="$DOCKER" bash tools/selfhost/compose.sh "$@"; }
hostpath() { # a path the Docker command understands
  if command -v wslpath >/dev/null && [[ "$DOCKER" == *.exe ]]; then wslpath -w "$1"; else realpath "$1"; fi
}
die() { echo "selfhost: $*" >&2; exit 1; }

case "${1:-up}" in
stop) compose stop; exit 0 ;;
clean)
  compose --profile setup --profile update --profile bundled --profile dev down -v
  # The demo's stand-in off-box copy was written by the jobs container, so remove it through one.
  if [ -d "$DIR/.offsite" ]; then
    "$DOCKER" run --rm -u 0 -v "$(hostpath "$(cd "$DIR/.offsite" && pwd)"):/o" "$HELPER" sh -c 'rm -rf /o/* /o/.[!.]*' >/dev/null
  fi
  rm -rf "$DIR"; exit 0 ;;
up) ;;
*) echo "usage: selfhost.sh [up|stop|clean]" >&2; exit 2 ;;
esac

# --- The configuration: the install's file, or the demo's defaults.
DEMO=1
if [ -n "${SB_CONFIG:-}" ]; then
  [ -f "$SB_CONFIG" ] || die "no configuration at $SB_CONFIG"
  # shellcheck disable=SC1090
  set -a; . "$SB_CONFIG"; set +a
  DEMO=
fi
SB_TENANT=${SB_TENANT:-acme-logistics}
SB_HOST=${SB_HOST:-cell.localhost}
SB_PORT=${SB_PORT:-8443}
SB_HTTP_PORT=${SB_HTTP_PORT:-8180}
SB_BIND=${SB_BIND:-127.0.0.1}
SB_TLS=${SB_TLS:-internal}
SB_IDP=${SB_IDP:-bundled}
SB_OFFSITE=${SB_OFFSITE:-}
SB_SHARE=${SB_SHARE:-}
SB_DATABASE=${SB_DATABASE:-bundled}
SB_HTTPS_PROXY=${SB_HTTPS_PROXY:-}
SB_HTTPS_PROXY_FILE=${SB_HTTPS_PROXY_FILE:-}
SB_SEAL=${SB_SEAL:-on}
case "$SB_SEAL" in on | off) ;; *) die "SB_SEAL is on or off" ;; esac
sbcli() { bash tools/selfhost/sb.sh "$@"; }
[[ "$SB_TENANT" =~ ^[a-z0-9][a-z0-9-]{1,62}$ ]] || die "SB_TENANT must be lower-case letters, digits and dashes"
[[ "$SB_HOST" =~ ^[a-z0-9.-]+$ ]] || die "SB_HOST must be a host name"
if [ "$SB_PORT" = 443 ]; then BASE=https://$SB_HOST; else BASE=https://$SB_HOST:$SB_PORT; fi
case "$SB_TLS" in
internal | acme) ;;
files) [ -f "${SB_TLS_CERT:-}" ] && [ -f "${SB_TLS_KEY:-}" ] || die "SB_TLS=files needs SB_TLS_CERT and SB_TLS_KEY" ;;
*) die "SB_TLS is internal, acme or files" ;;
esac
case "$SB_IDP" in
bundled) ISSUER=$BASE/idp CLIENT=sb-$SB_TENANT ;;
company)
  [ -n "${SB_OIDC_ISSUER:-}" ] && [ -n "${SB_OIDC_CLIENT_ID:-}" ] && [ -f "${SB_OIDC_SECRET_FILE:-}" ] ||
    die "SB_IDP=company needs SB_OIDC_ISSUER, SB_OIDC_CLIENT_ID and SB_OIDC_SECRET_FILE"
  [[ "$SB_OIDC_ISSUER" == https://* ]] || die "the company's issuer must be https://"
  ISSUER=$SB_OIDC_ISSUER CLIENT=$SB_OIDC_CLIENT_ID ;;
dev)
  # The development IdP has no passwords: never where anyone but this machine can reach it.
  [ "$SB_BIND" = 127.0.0.1 ] || die "SB_IDP=dev has no passwords and is refused unless SB_BIND=127.0.0.1"
  ISSUER=http://idp.localhost:8081 CLIENT=sb-$SB_TENANT ;;
*) die "SB_IDP is bundled, company or dev" ;;
esac
if [ -n "$DEMO" ]; then
  SB_SHARE=${SB_SHARE:-examples/share}
  SB_OFFSITE=${SB_OFFSITE:-$DIR/.offsite}
else
  [ -f "${SB_MANIFEST:-}" ] || die "SB_MANIFEST must name the tenant's manifest (sb-tenant(5))"
  [[ "$SB_OFFSITE" == s3://* ]] || { [ -n "$SB_OFFSITE" ] && [ -d "$SB_OFFSITE" ]; } || die "SB_OFFSITE must name a directory on another machine, mounted here, or s3://BUCKET/PREFIX (the off-box backup copy)"
  [ -n "${SB_RELEASE_STORE:-}" ] && [ -n "${SB_RELEASE_TRUSTED:-}" ] || die "SB_RELEASE_STORE and SB_RELEASE_TRUSTED name where releases come from and who signs them"
fi
# The outbound proxy (SB-1806): a URL without credentials, or a file holding one with them.
[ -z "$SB_HTTPS_PROXY" ] || [ -z "$SB_HTTPS_PROXY_FILE" ] || die "set SB_HTTPS_PROXY or SB_HTTPS_PROXY_FILE, not both"
if [ -n "$SB_HTTPS_PROXY" ]; then sbcli selfhost check-proxy "$SB_HTTPS_PROXY" >/dev/null || die "SB_HTTPS_PROXY is refused (above)"; fi
if [ -n "$SB_HTTPS_PROXY_FILE" ]; then
  [ -f "$SB_HTTPS_PROXY_FILE" ] || die "SB_HTTPS_PROXY_FILE ($SB_HTTPS_PROXY_FILE) isn't a file"
  sbcli selfhost check-proxy --file "$SB_HTTPS_PROXY_FILE" >/dev/null || die "SB_HTTPS_PROXY_FILE is refused (above)"
fi
# The company's bucket for the off-box copies (SB-2303): sealed always, its key pair in a
# file, checked before anything changes.
OFFSITE_S3=
if [[ "$SB_OFFSITE" == s3://* ]]; then
  OFFSITE_S3=$SB_OFFSITE
  [ "$SB_SEAL" = on ] || die "SB_OFFSITE is a bucket: nothing unsealed leaves the box, so SB_SEAL=off is refused"
  [ -f "${SB_OFFSITE_KEY_FILE:-}" ] || die "SB_OFFSITE=s3://… needs SB_OFFSITE_KEY_FILE, a file holding the bucket's key pair"
  sbcli backup check --to "$OFFSITE_S3" --keys "$SB_OFFSITE_KEY_FILE" >/dev/null || die "the bucket $OFFSITE_S3 doesn't answer with that key pair (sb backup check)"
  SB_OFFSITE=$DIR/.offsite-s3 # the job's mount point, unused: the copy goes to the bucket
fi
# The database (SB-1807): the bundled one, or the company's on another machine.
case "$SB_DATABASE" in
bundled) ;;
external)
  [ -f "${SB_DATABASE_DSN_FILE:-}" ] || die "SB_DATABASE=external needs SB_DATABASE_DSN_FILE, a file holding the connection string"
  sbcli selfhost check-dsn "$SB_DATABASE_DSN_FILE" >/dev/null || die "SB_DATABASE_DSN_FILE is refused (above)" ;;
*) die "SB_DATABASE is bundled or external" ;;
esac
abs() { mkdir -p "$1" && (cd "$1" && pwd); }

rand() { head -c 32 /dev/urandom | base64 | tr -d '\n/+=' | head -c 40; }

[ "${SKIP_IMAGES:-}" = 1 ] || DOCKER="$DOCKER" bash tools/selfhost/images.sh >/dev/null

# --- Secrets and keys: made on this machine, once, and never anywhere else.
V=$DIR/vault
# The vaults and cell.json as they were, so a change to either recreates the services below.
confsum() { [ -d "$V" ] && { find "$V" -type f -exec sha256sum {} +; sha256sum "$DIR/cell.json" 2>/dev/null; } | sort | sha256sum; }
CONF_BEFORE=$(confsum || true)
# Every vault file owner-only (SB-2604): sb cell check-config refuses one others can read.
put() { (umask 077; mkdir -p "$(dirname "$V/$1/vault/tenants/$SB_TENANT/$2")"; printf '%s' "$3" >"$V/$1/vault/tenants/$SB_TENANT/$2"); }
mkdir -p "$DIR/policy/platform" "$DIR/tls" "$DIR/ca" "$DIR/releases" "$DIR/iam" "$V/updater/vault/tenants/$SB_TENANT"
if [ ! -f "$DIR/db-password" ]; then
  dbpw=$(rand)
  printf '%s' "$dbpw" >"$DIR/db-password"
  oidc=$(rand) scim=$(rand) cp=$(rand)
  anchor=$(head -c 32 /dev/urandom | base64 | tr -d '\n') attest=$(head -c 32 /dev/urandom | base64 | tr -d '\n')
  dsn="postgres://sb:$dbpw@postgres:5432/sb?sslmode=disable"
  # Each service's vault holds only what that service reads (sb-selfhost(7), The layout).
  for f in "oidc $oidc" "scim $scim" "control-plane $cp" "audit/anchor-seed $anchor" "attest-seed $attest" "database $dsn"; do put api ${f%% *} "${f#* }"; done
  put ingest database "$dsn"
  for f in "audit/anchor-seed $anchor" "database $dsn"; do put jobs ${f%% *} "${f#* }"; done
fi
if [ "$SB_IDP" = company ]; then put api oidc "$(cat "$SB_OIDC_SECRET_FILE")"; fi
# The database each service reads: the company's (SB-1807), or the bundled one, written
# every run so that moving back from external to bundled takes effect too.
rm -f "$DIR/ca/database-ca.crt"
if [ "$SB_DATABASE" = external ]; then
  dsn=$(tr -d '\r\n' <"$SB_DATABASE_DSN_FILE")
  if [ -n "${SB_DATABASE_CA:-}" ]; then cp "$SB_DATABASE_CA" "$DIR/ca/database-ca.crt"; fi
else
  dsn="postgres://sb:$(cat "$DIR/db-password")@postgres:5432/sb?sslmode=disable"
fi
for svc in api ingest jobs; do put $svc database "$dsn"; done
PROXY_FILE_IN=
for svc in api ingest jobs updater; do rm -f "$V/$svc/vault/tenants/$SB_TENANT/https-proxy"; done
if [ -n "$SB_HTTPS_PROXY_FILE" ]; then
  for svc in api ingest jobs updater; do put $svc https-proxy "$(tr -d '\r\n' <"$SB_HTTPS_PROXY_FILE")"; done
  PROXY_FILE_IN=/var/lib/sb/vault/vault/tenants/$SB_TENANT/https-proxy
fi
NO_PROXY_LIST=
if [ -n "$SB_HTTPS_PROXY$SB_HTTPS_PROXY_FILE" ]; then
  # The cell's own names never go through the proxy: its sign-in, its services, loopback.
  NO_PROXY_LIST=$SB_HOST,localhost,127.0.0.1,::1,api,idp,worm,model,model-embed${SB_NO_PROXY:+,$SB_NO_PROXY}
fi
if [ -n "${SB_RELEASE_PULL_TOKEN_FILE:-}" ]; then put updater release-pull "$(cat "$SB_RELEASE_PULL_TOKEN_FILE")"; fi
# The write-once store's key (SB-1603): the store reads the secret, the jobs service the pair.
if [ ! -f "$DIR/worm-secret" ]; then
  wormsecret=$(rand)
  printf '%s' "$wormsecret" >"$DIR/worm-secret"
  put jobs worm "$(printf '{"access_key":"sb-worm","secret_key":"%s"}' "$wormsecret")"
fi

# The mail server's sign-in (SB-2601): the services that send, api and jobs, only.
for svc in api jobs; do rm -f "$V/$svc/vault/tenants/$SB_TENANT/email"; done
if [ -n "${SB_EMAIL_LOGIN_FILE:-}" ]; then for svc in api jobs; do put $svc email "$(tr -d '\r' <"$SB_EMAIL_LOGIN_FILE")"; done; fi
[ ! -d "$V" ] || chmod -R go= "$V"

# The bucket's key pair (SB-2303): the jobs service's vault only, read by sb on the host.
rm -f "$V/jobs/vault/tenants/$SB_TENANT/offsite"
if [ -n "$OFFSITE_S3" ]; then put jobs offsite "$(tr -d '\r' <"$SB_OFFSITE_KEY_FILE")"; fi

# The recovery key (SB-1901): made once, its public half kept here to seal every off-box
# copy to; the private half waits for the recovery kit (install.sh --recovery-kit), which
# takes it off the box.
SEAL=
if [ "$SB_SEAL" = on ]; then
  [ -f "$DIR/recovery.pub" ] || sbcli backup keygen --public "$DIR/recovery.pub" --private "$DIR/recovery-key.private" >/dev/null
  SEAL=$(tr -d '\r\n' <"$DIR/recovery.pub")
else
  echo "selfhost: SB_SEAL=off: off-box copies are not sealed; whoever reads SB_OFFSITE reads them" >&2
  rm -f "$DIR/recovery.pub"
fi

# --- Policies and the manifest.
cp policy/platform/*.cedar "$DIR/policy/platform/"
if [ -n "${SB_POLICY:-}" ]; then cp "$SB_POLICY" "$DIR/policy/$SB_TENANT.cedar"
elif [ -f "examples/policies/$SB_TENANT.cedar" ]; then cp "examples/policies/$SB_TENANT.cedar" "$DIR/policy/"; fi
if [ -n "$DEMO" ]; then
  cat >"$DIR/tenant.yaml" <<YAML
tenant: $SB_TENANT
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask, draft-reply]
retention_days: 365
budget_usd_month: 1500
YAML
else
  cp "$SB_MANIFEST" "$DIR/tenant.yaml"
  grep -qE "^tenant: *$SB_TENANT *$" "$DIR/tenant.yaml" || die "the manifest's tenant isn't $SB_TENANT"
fi

# --- The release the updater admits. The demo signs this machine's own images with a key
# made for this machine (SB-1604); an install trusts the factory's release identity.
if [ -n "$DEMO" ]; then
  digest() { sed -nE "s/^$1 (sha256:[0-9a-f]{64})\$/\1/p" build/images/digests.txt | tr -d '\r'; }
  echo "demo release of $(git rev-parse --short HEAD 2>/dev/null): make selfhost" >"$DIR/gate-results.txt"
  go run ./tools/release/bundle --image api="$(digest api)" --image ingest="$(digest ingest)" --image jobs="$(digest jobs)" \
    --image updater="$(digest updater)" --created "$(date -u +%F)" --out "$DIR/bundle.json" >/dev/null
  read -r _ TRUSTED < <(go run ./tools/selfhost/devrelease --bundle "$DIR/bundle.json" --store "$DIR/releases" \
    --key-file "$DIR/release-key" --gate-results "$DIR/gate-results.txt")
  STORE=file:///var/lib/sb/releases REGISTRY=local
else
  TRUSTED=$SB_RELEASE_TRUSTED REGISTRY=${SB_RELEASE_REGISTRY:-local}
  case "$SB_RELEASE_STORE" in
  https://*) STORE=$SB_RELEASE_STORE ;;
  *) [ -d "$SB_RELEASE_STORE" ] || die "SB_RELEASE_STORE is an https:// store or a directory of releases"
     cp -a "$SB_RELEASE_STORE/." "$DIR/releases/"; STORE=file:///var/lib/sb/releases ;;
  esac
fi

# --- What the factory can reach (LAW-17 on one machine): at most the release store's pull
# token, read-only. The monthly access job checks this list and records a statement.
if [[ "$STORE" == https://* ]]; then
  printf '[{"kind":"pull-token","id":"release-pull","grant":"release-pull"}]\n' >"$DIR/iam/$SB_TENANT.json"
  RETAIN='["release-pull"]'
else
  printf '[]\n' >"$DIR/iam/$SB_TENANT.json"
  RETAIN='[]'
fi

# --- The cell's configuration (sb-cell(5)). With a gated model on the box (build/selfhost/
# model.json, written by model.sh only after the eval gate passed), the model profile.
MODEL='{ "provider": "extractive" }'
EMBED='{ "provider": "hashing" }'
MODEL_IMAGE=
if [ -f "$DIR/model.json" ]; then
  mf() { sed -nE "s/^ *\"$1\": *\"?([^\",]*)\"?,?\$/\1/p" "$DIR/model.json"; }
  MODEL="{ \"provider\": \"openaicompat\", \"base\": \"http://model:8080/v1\", \"id\": \"$(mf model)\", \"local\": true }"
  EMBED="{ \"provider\": \"openaicompat\", \"base\": \"http://model-embed:8080/v1\", \"model\": \"$(mf embed_model)\", \"dim\": $(mf embed_dim) }"
  MODEL_IMAGE=$(mf image)
fi
# SB-6102: with a transcriber on the box (build/selfhost/transcribe.json, written by
# transcribe.sh only after its sample was heard), the transcribe profile.
TRANSCRIBE_IMAGE=
if [ -f "$DIR/transcribe.json" ]; then
  TRANSCRIBE_IMAGE=$(sed -nE 's/^ *"image": *"([^"]*)".*$/\1/p' "$DIR/transcribe.json")
  MODEL="${MODEL%\}}, \"transcribe\": { \"provider\": \"whispercpp\", \"base\": \"http://transcribe:8080\", \"local\": true } }"
fi
CONNECTORS='{}'
if [ -n "$SB_SHARE" ]; then CONNECTORS='{ "docs": { "base": "file:///srv/share" } }'; fi
# The backup's off-box copy: the mounted directory, or the bucket (always sealed).
if [ -n "$OFFSITE_S3" ]; then
  OFFSITE_COPY="{ \"s3\": \"$OFFSITE_S3\", \"offsite\": true, \"seal\": \"$SEAL\" }"
else
  OFFSITE_COPY="{ \"dir\": \"/srv/backup-offsite\", \"media\": \"disk\", \"offsite\": true${SEAL:+, \"seal\": \"$SEAL\"} }"
fi
EMAIL=
if [ -n "${SB_EMAIL_SERVER:-}" ]; then EMAIL="
  \"email\": { \"server\": \"$SB_EMAIL_SERVER\", \"from\": \"$SB_EMAIL_FROM\" },"; fi
cat >"$DIR/cell.json" <<JSON
{
  "tenant": "$SB_TENANT",
  "manifest": "/var/lib/sb/state/tenant.yaml",
  "state": "/var/lib/sb/state",
  "vault": "/var/lib/sb/vault",
  "policies": "/etc/sb/policy",
  "listen": { "api": "0.0.0.0:8080" },
  "base_url": "$BASE",
  "oidc": { "client_id": "$CLIENT", "issuer": "$ISSUER" },
  "web": "/sb/web",
  "model": $MODEL,
  "embed": $EMBED,
  "database": "vault://tenants/$SB_TENANT/database",
  "connectors": $CONNECTORS,
  "worm": { "endpoint": "http://worm:7070", "bucket": "sb-worm" },$EMAIL
  "backups": [
    { "dir": "/var/lib/sb/state/backups", "media": "disk" },
    { "worm": true },
    $OFFSITE_COPY
  ],
  "custody": { "tenant": true, "retain": $RETAIN },
  "release": { "store": "$STORE", "registry": "$REGISTRY", "trusted": "$TRUSTED", "deploy": { "kind": "local" } },
  "durable": true
}
JSON

# --- The proxy: the cell, the bundled sign-in at /idp, and the install's TLS.
{
  echo "{"
  case "$SB_TLS" in
  internal) printf '\tlocal_certs\n\tauto_https disable_redirects\n' ;;
  acme) if [ -n "${SB_ACME_EMAIL:-}" ]; then printf '\temail %s\n' "$SB_ACME_EMAIL"; fi ;;
  files) printf '\tauto_https disable_certs\n' ;;
  esac
  echo "}"
  echo
  echo "$SB_HOST:$SB_PORT {"
  if [ "$SB_TLS" = files ]; then printf '\ttls /etc/sb/tls/cert.pem /etc/sb/tls/key.pem\n'; fi
  printf '\theader Strict-Transport-Security "max-age=31536000"\n'
  if [ "$SB_IDP" = bundled ]; then printf '\thandle /idp/* {\n\t\treverse_proxy idp:8082\n\t}\n'; fi
  printf '\thandle {\n\t\treverse_proxy api:8080\n\t}\n'
  echo "}"
} >"$DIR/Caddyfile"
if [ "$SB_TLS" = files ]; then
  cp "$SB_TLS_CERT" "$DIR/tls/cert.pem"; cp "$SB_TLS_KEY" "$DIR/tls/key.pem"; chmod 600 "$DIR/tls/key.pem"
  if [ -n "${SB_TLS_CA:-}" ]; then cp "$SB_TLS_CA" "$DIR/ca/company-ca.crt"; fi
fi

# --- compose.env carries the layout's settings; the admitted image digests stay.
profiles=$SB_IDP
[ "$SB_IDP" = company ] && profiles=
[ -z "$MODEL_IMAGE" ] || profiles="$profiles model"
[ -z "$TRANSCRIBE_IMAGE" ] || profiles="$profiles transcribe"
[ -n "$SB_SHARE" ] || mkdir -p "$DIR/empty-share/notes" # the notes mount point (SB-1705)
{
  echo "SB_TENANT=$SB_TENANT"; echo "SB_HOST=$SB_HOST"; echo "SB_PORT=$SB_PORT"; echo "SB_HTTP_PORT=$SB_HTTP_PORT"
  echo "SB_BIND=$SB_BIND"; echo "SB_BASE_URL=$BASE"; echo "SB_ISSUER=$ISSUER"
  echo "SB_SHARE=$(hostpath "$(abs "${SB_SHARE:-$DIR/empty-share}")")"; echo "SB_OFFSITE=$(hostpath "$(abs "$SB_OFFSITE")")"
  [ -z "$OFFSITE_S3" ] || echo "SB_OFFSITE_S3=$OFFSITE_S3"
  [ -z "${SB_NOTES:-}" ] || echo "SB_NOTES=$(hostpath "$(abs "$SB_NOTES")")"
  echo "SB_PROFILES=$profiles"
  echo "SB_DATABASE=$SB_DATABASE"
  [ -z "$SB_HTTPS_PROXY" ] || echo "SB_HTTPS_PROXY=$SB_HTTPS_PROXY"
  [ -z "$PROXY_FILE_IN" ] || echo "SB_PROXY_FILE=$PROXY_FILE_IN"
  [ -z "$NO_PROXY_LIST" ] || echo "SB_NO_PROXY_LIST=$NO_PROXY_LIST"
  [ -z "$MODEL_IMAGE" ] || echo "SB_MODEL_IMAGE=$MODEL_IMAGE"
  [ -z "$TRANSCRIBE_IMAGE" ] || echo "SB_TRANSCRIBE_IMAGE=$TRANSCRIBE_IMAGE"
  [ -z "${SB_EMAIL_SERVER:-}" ] || echo "SB_EMAIL_SERVER=$SB_EMAIL_SERVER"
  if [ -f "$DIR/compose.env" ]; then grep -E '^SB_[A-Z]+_IMAGE=' "$DIR/compose.env" || true; fi
} >"$DIR/compose.env.new"
mv "$DIR/compose.env.new" "$DIR/compose.env"

# --- Fill the volumes: each vault owned by the service user (65532) with no access for others.
for v in vault-api vault-ingest vault-jobs vault-updater releases state idp-data run-idp run-wormsetup caddy worm-data pgdata; do
  "$DOCKER" volume inspect "sb-selfhost_$v" >/dev/null 2>&1 ||
    "$DOCKER" volume create --label com.docker.compose.project=sb-selfhost --label com.docker.compose.volume=$v "sb-selfhost_$v" >/dev/null
done
fill() { # volume source-dir
  "$DOCKER" run --rm -u 0 -v "sb-selfhost_$1:/dst" -v "$(hostpath "$2"):/src:ro" "$HELPER" \
    sh -c 'cp -a /src/. /dst/ && chown -R 65532:65532 /dst && chmod -R u=rwX,go= /dst' >/dev/null
}
for svc in api ingest jobs updater; do fill "vault-$svc" "$V/$svc"; done
fill releases "$DIR/releases"
"$DOCKER" run --rm -u 0 -v sb-selfhost_state:/dst -v "$(hostpath "$DIR"):/src:ro" "$HELPER" \
  sh -c 'cp /src/tenant.yaml /dst/tenant.yaml && mkdir -p /dst/iam /dst/backups && cp /src/iam/*.json /dst/iam/ &&
         chown -R 65532:65532 /dst && chmod 700 /dst' >/dev/null
"$DOCKER" run --rm -u 0 -v sb-selfhost_idp-data:/dst "$HELPER" sh -c 'chown 65532:65532 /dst && chmod 700 /dst' >/dev/null
# The one-file secrets of the IdP and wormsetup: vault files are owner-only on the host
# (SB-2604), so a bind mount is unreadable to 65532 unless the host user happens to be
# 65532. Copy each into the service's own volume, owned by 65532, no access for others.
seal() { # volume name=vault-file...
  local vol=$1 p; shift
  local mounts=()
  for p; do [ ! -f "${p#*=}" ] || mounts+=(-v "$(hostpath "${p#*=}"):/src/${p%%=*}:ro"); done
  "$DOCKER" run --rm -u 0 -v "sb-selfhost_$vol:/dst" "${mounts[@]}" "$HELPER" \
    sh -c 'rm -rf /dst/* && { [ ! -d /src ] || cp /src/* /dst/; } && chown -R 65532:65532 /dst && chmod 700 /dst && chmod -R go= /dst' >/dev/null
}
IDPV=$V/api/vault/tenants/$SB_TENANT
seal run-idp "oidc=$IDPV/oidc" "scim=$IDPV/scim"
seal run-wormsetup "worm-creds=$V/jobs/vault/tenants/$SB_TENANT/worm"
# The demo's stand-in off-box directory: the jobs service (65532) writes the backup's copy
# there, as install.sh arranges for a real SB_OFFSITE.
if [ "$SB_OFFSITE" = "$DIR/.offsite" ]; then
  mkdir -p "$SB_OFFSITE"
  "$DOCKER" run --rm -u 0 -v "$(hostpath "$SB_OFFSITE"):/dst" "$HELPER" sh -c 'chown 65532:65532 /dst && chmod 755 /dst' >/dev/null
fi
# Stewards' notes (SB-1705): written by api alone, read by ingest; on a real install SB_NOTES
# names the share's own notes/ folder, given to 65532 by install.sh.
[ -n "${SB_NOTES:-}" ] || "$DOCKER" run --rm -u 0 -v sb-selfhost_notes:/dst "$HELPER" sh -c 'chown 65532:65532 /dst && chmod 755 /dst' >/dev/null

# A changed vault or cell.json (another database, a proxy set or cleared, sealing turned on)
# reaches running services only when they start again; they are recreated after the
# layout is up.
CONF_AFTER=$(confsum || true)
RECREATE=
[ -z "$CONF_BEFORE" ] || [ "$CONF_BEFORE" = "$CONF_AFTER" ] || RECREATE=1

# --- Up: the proxy first, so the cell can trust its local CA before it looks for its IdP.
compose up -d proxy >/dev/null 2>&1 || compose up -d proxy
if [ "$SB_TLS" = internal ]; then
  for _ in $(seq 30); do
    if "$DOCKER" run --rm -v sb-selfhost_caddy:/c:ro "$HELPER" test -s /c/caddy/pki/authorities/local/root.crt; then break; fi
    sleep 1
  done
  "$DOCKER" run --rm -v sb-selfhost_caddy:/c:ro "$HELPER" cat /c/caddy/pki/authorities/local/root.crt | tr -d '\r' >"$DIR/ca/caddy-root.crt"
  [ -s "$DIR/ca/caddy-root.crt" ] || die "the proxy made no local CA; 'tools/selfhost/compose.sh logs proxy' says why"
fi
compose up -d >/dev/null 2>&1 || compose up -d
if [ -n "$RECREATE" ]; then compose up -d --force-recreate api ingest jobs >/dev/null 2>&1 || compose up -d --force-recreate api ingest jobs; fi
# A database on another machine (SB-1807): the bundled one stops; its volume stays.
if [ "$SB_DATABASE" = external ]; then compose --profile db rm -sf postgres >/dev/null 2>&1 || true; fi
# The write-once bucket: created once, locked for seven years, and proved by a probe.
compose --profile setup run --rm wormsetup >/dev/null 2>&1 || compose --profile setup run --rm wormsetup
ready() {
  for _ in $(seq 90); do
    if "$CURL" -ksf --resolve "$SB_HOST:$SB_PORT:127.0.0.1" "$BASE/readyz" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  return 1
}
ok=
if ready; then ok=1
elif grep -qE '^SB_API_IMAGE=' "$DIR/compose.env" && compose logs --tail 20 api 2>/dev/null | grep -q 'unknown field'; then
  # The admitted release predates a setting this install writes (sb-selfhost(7), Upgrades):
  # start the install's own images, as a first install does, keeping what ran for one
  # rollback; update.sh (which install.sh runs next) pins the admitted release again.
  echo "selfhost: the running release predates this install's settings; starting this install's images" >&2
  prev=
  for svc in api ingest jobs updater; do
    cur=$(sed -nE "s/^SB_${svc^^}_IMAGE=//p" "$DIR/compose.env" | tr -d '\r')
    run=$("$DOCKER" run --rm -v sb-selfhost_state:/s:ro "$HELPER" cat "/s/deploy/running/$svc.json" 2>/dev/null | tr -d '\r' || true)
    ref=$(printf '%s' "$run" | sed -nE 's/.*"image":"([^"]+)".*/\1/p'); rel=$(printf '%s' "$run" | sed -nE 's/.*"release":"([^"]+)".*/\1/p')
    [ -n "$cur" ] && [ -n "$ref" ] || { prev=; break; }
    prev+="SB_${svc^^}_IMAGE=$cur"$'\n'"REF_$svc=$ref"$'\n'"REL_$svc=$rel"$'\n'
  done
  [ -z "$prev" ] || printf '%s' "$prev" >"$DIR/previous.env"
  grep -vE '^SB_(API|INGEST|JOBS|UPDATER)_IMAGE=' "$DIR/compose.env" >"$DIR/compose.env.new" || true
  mv "$DIR/compose.env.new" "$DIR/compose.env"
  compose up -d >/dev/null 2>&1 || compose up -d
  ready && ok=1
fi
[ -n "$ok" ] || die "the cell didn't become ready; 'tools/selfhost/compose.sh logs api' says why"

# --- The demo's user, provisioned the way a company's IdP would: over SCIM.
if [ -n "$DEMO" ]; then
  if [ "$SB_IDP" = bundled ]; then
    if [ ! -f "$DIR/demo-password" ]; then
      pw=$(rand | head -c 20)
      if out=$(printf '%s' "$pw" | compose run --rm -T idp user add --data /var/lib/sb-idp --email ann@acme.example \
        --scim http://api:8080/scim/v2 --scim-token-file /run/sb/scim 2>/dev/null); then
        printf '%s' "$pw" >"$DIR/demo-password"
        # The demo's one user is its operator too, so the console (health, the unhelpful-
        # answer queue) can be tried: sb-operators, over SCIM like any group (sb-admin(3)).
        id=$(printf '%s' "$out" | sed -nE 's/.* as (usr_[A-Za-z0-9_]+).*/\1/p' | tr -d '\r')
        scim=$(cat "$V/api/vault/tenants/$SB_TENANT/scim")
        [ -n "$id" ] && "$CURL" -ksf --resolve "$SB_HOST:$SB_PORT:127.0.0.1" -X POST "$BASE/scim/v2/Groups" -H "Authorization: Bearer $scim" \
          -H "Content-Type: application/scim+json" -d "{\"displayName\":\"sb-operators\",\"members\":[{\"value\":\"$id\"}]}" >/dev/null ||
          echo "selfhost: ann couldn't be made the demo's operator; the console will say so" >&2
      else
        die "the demo user couldn't be added; 'tools/selfhost/compose.sh run --rm idp user list --data /var/lib/sb-idp' shows who exists"
      fi
    fi
    echo "selfhost: $SB_TENANT is running at $BASE"
    echo "selfhost: sign in at $BASE/login as ann@acme.example with the password in $DIR/demo-password; $BASE/?admin is the console"
  else
    scim=$(cat "$V/api/vault/tenants/$SB_TENANT/scim")
    body=$("$CURL" -ks --resolve "$SB_HOST:$SB_PORT:127.0.0.1" -X POST "$BASE/scim/v2/Users" -H "Authorization: Bearer $scim" \
      -H "Content-Type: application/scim+json" -d '{"userName":"ann","emails":[{"value":"ann@acme.example"}]}')
    id=$(printf '%s' "$body" | sed -nE 's/.*"id":"(usr_[0-9a-f]+)".*/\1/p')
    if [ -n "$id" ]; then printf '%s' "$id" >"$DIR/demo-user"; else id=$(cat "$DIR/demo-user" 2>/dev/null || true); fi
    [ -n "$id" ] && "$CURL" -s -X POST "http://idp.localhost:8081/dev/user?id=$id" --resolve idp.localhost:8081:127.0.0.1 >/dev/null
    echo "selfhost: $SB_TENANT is running at $BASE; /login signs in as ann ($id) with no password (SB_IDP=dev)"
  fi
  echo "selfhost: make selfhost-update moves it onto the admitted release; make selfhost-stop stops it, make selfhost-clean deletes its data"
else
  echo "selfhost: $SB_TENANT is running at $BASE"
fi
