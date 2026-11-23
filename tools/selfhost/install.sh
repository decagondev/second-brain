#!/usr/bin/env bash
# install.sh (sb-selfhost(7), Install; SB-1605): a self-hosted cell on one Linux VM or
# on-prem server, from a checkout or an install bundle (make selfhost-bundle).
#
#   sudo tools/selfhost/install.sh [--install-docker] [--check] CONFIG
#   sudo tools/selfhost/install.sh --doctor
#   sudo tools/selfhost/install.sh --support
#   sudo tools/selfhost/install.sh --recovery-kit [--new-key] [--custodian N=NAME]...
#   sudo tools/selfhost/install.sh --retire [CONFIG] --yes-destroy
#
#   --install-docker   install Docker Engine and the compose plugin from the distribution
#                      (Debian or Ubuntu) when they are missing; otherwise they must be there
#   --check            check the machine and CONFIG, change nothing
#   --doctor           check the running box and say how to fix each problem (SB-1801)
#   --support          write a support bundle with no content and no secret (SB-1808)
#   --recovery-kit     take the recovery key off the box into 5 shares, any 3 of which
#                      rebuild it (SB-1902); the rest of the arguments go to kit.sh
#   --retire           destroy the box for good and write a signed statement of what was
#                      destroyed (SB-1907); refuses without --yes-destroy (retire.sh)
#
# It loads the bundle's images (or builds them, in a checkout with Go), runs selfhost.sh
# with CONFIG, applies the first release, installs the systemd units (the cell at boot,
# hourly upgrades, nightly database dumps), and says how to add the first user and run
# sb tenant verify. Every key and secret is made on this machine and stays on it.
set -euo pipefail
cd "$(dirname "$0")/../.."
HOME_DIR=$(pwd)
die() { echo "install: $*" >&2; exit 1; }
INSTALL_DOCKER= CHECK= CONFIG= DOCTOR= SUPPORT=
if [ "${1:-}" = --recovery-kit ]; then shift; exec bash tools/selfhost/kit.sh "$@"; fi
if [ "${1:-}" = --retire ]; then shift; exec bash tools/selfhost/retire.sh "$@"; fi
for a in "$@"; do
  case "$a" in
  --install-docker) INSTALL_DOCKER=1 ;;
  --check) CHECK=1 ;;
  --doctor) DOCTOR=1 ;;
  --support) SUPPORT=1 ;;
  -*) die "unknown option $a" ;;
  *) CONFIG=$a ;;
  esac
done
# The running box, judged by sb selfhost doctor (internal/core/box) over facts.sh's facts.
if [ -n "$DOCTOR" ]; then
  [ -f build/selfhost/compose.env ] || die "no self-hosted cell here to check"
  rc=0
  bash tools/selfhost/facts.sh | bash tools/selfhost/sb.sh selfhost doctor || rc=$?
  # The mail server (SB-2601), when the box sends notifications by email.
  srv=$(sed -nE 's/^SB_EMAIL_SERVER=//p' build/selfhost/compose.env | tr -d '\r')
  if [ -n "$srv" ]; then
    ca=; [ ! -f build/selfhost/ca/company-ca.crt ] || ca=build/selfhost/ca/company-ca.crt
    bash tools/selfhost/sb.sh selfhost check-email "$srv" ${ca:+--ca "$ca"} || rc=1
  fi
  exit $rc
fi
if [ -n "$SUPPORT" ]; then exec bash tools/selfhost/support.sh; fi
[ -n "$CONFIG" ] || die "usage: install.sh [--install-docker] [--check] CONFIG (see deploy/selfhost/install.env.example)"
[ -f "$CONFIG" ] || die "no configuration at $CONFIG"
CONFIG=$(cd "$(dirname "$CONFIG")" && pwd)/$(basename "$CONFIG")
[ "$(uname -s)" = Linux ] || die "install.sh is for a Linux machine; on a laptop, make selfhost runs the demo"
[ "$(id -u)" = 0 ] || die "run it as root (sudo): it installs systemd units and owns the cell's directories"

# The machine.
if ! command -v docker >/dev/null || ! docker compose version >/dev/null 2>&1; then
  [ -n "$INSTALL_DOCKER" ] || die "Docker Engine and the compose plugin are needed; install them, or pass --install-docker"
  [ -z "$CHECK" ] || die "Docker is missing (--install-docker would install it)"
  command -v apt-get >/dev/null || die "--install-docker knows Debian and Ubuntu only; install Docker yourself"
  apt-get update -q && apt-get install -y -q docker.io docker-compose-v2 || apt-get install -y -q docker.io docker-compose-plugin
  systemctl enable --now docker
fi
command -v systemctl >/dev/null || die "systemd is needed for the cell's timers"
for t in curl sha256sum base64; do command -v $t >/dev/null || die "$t is needed"; done

# The configuration, checked the way selfhost.sh will read it.
# shellcheck disable=SC1090
set -a; . "$CONFIG"; set +a
[ "${SB_IDP:-bundled}" != dev ] || die "SB_IDP=dev has no passwords; an install uses company or bundled"
[ "${SB_TLS:-internal}" != internal ] || echo "install: SB_TLS=internal: every browser must trust Caddy's own CA; use acme or files for people" >&2
if [[ "${SB_OFFSITE:-}" == s3://* ]]; then # the company's bucket (SB-2303)
  [ "${SB_SEAL:-on}" = on ] || die "SB_OFFSITE is a bucket: nothing unsealed leaves the box, so SB_SEAL=off is refused"
  [ -f "${SB_OFFSITE_KEY_FILE:-}" ] || die "SB_OFFSITE=s3://… needs SB_OFFSITE_KEY_FILE, a file holding the bucket's key pair"
  o=$(bash tools/selfhost/sb.sh backup check --to "$SB_OFFSITE" --keys "$SB_OFFSITE_KEY_FILE") || die "the bucket doesn't answer with that key pair: $o"
  offsite="$o"
else
[ -d "${SB_OFFSITE:-/nonexistent}" ] || die "SB_OFFSITE ($SB_OFFSITE) must be a mounted directory on another machine, or s3://BUCKET/PREFIX"
offsite="$SB_OFFSITE"
if [ "$(findmnt -n -o TARGET -T "$SB_OFFSITE" 2>/dev/null || echo /)" = / ]; then
  echo "install: SB_OFFSITE ($SB_OFFSITE) is on this machine's root disk; the off-box copy must be on another machine" >&2
fi
fi
# The outbound proxy (SB-1806) and the database (SB-1807), checked and reported.
[ -z "${SB_HTTPS_PROXY:-}" ] || [ -z "${SB_HTTPS_PROXY_FILE:-}" ] || die "set SB_HTTPS_PROXY or SB_HTTPS_PROXY_FILE, not both"
if [ -n "${SB_HTTPS_PROXY:-}" ]; then
  p=$(bash tools/selfhost/sb.sh selfhost check-proxy "$SB_HTTPS_PROXY") || die "SB_HTTPS_PROXY is refused (above)"
  proxy="${p#proxy: }, no_proxy: ${SB_NO_PROXY:-the cell's own names only}"
elif [ -n "${SB_HTTPS_PROXY_FILE:-}" ]; then
  p=$(bash tools/selfhost/sb.sh selfhost check-proxy --file "$SB_HTTPS_PROXY_FILE") || die "SB_HTTPS_PROXY_FILE is refused (above)"
  proxy="${p#proxy: } (credentials from $SB_HTTPS_PROXY_FILE), no_proxy: ${SB_NO_PROXY:-the cell's own names only}"
else
  proxy="none: api, ingest, jobs and the updater reach the internet directly"
fi
# Notifications by email (SB-2601): the server answers with STARTTLS, and the sign-in is root's alone.
email="none"
if [ -n "${SB_EMAIL_SERVER:-}" ]; then
  [ -n "${SB_EMAIL_FROM:-}" ] || die "SB_EMAIL_SERVER needs SB_EMAIL_FROM, the sender address"
  if [ -n "${SB_EMAIL_LOGIN_FILE:-}" ]; then
    [ -f "$SB_EMAIL_LOGIN_FILE" ] || die "no SB_EMAIL_LOGIN_FILE at $SB_EMAIL_LOGIN_FILE"
    case "$(stat -c '%u %a' "$SB_EMAIL_LOGIN_FILE")" in "0 600" | "0 400") ;; *) die "SB_EMAIL_LOGIN_FILE must be owned by root with mode 0600" ;; esac
  fi
  e=$(bash tools/selfhost/sb.sh selfhost check-email "$SB_EMAIL_SERVER" ${SB_TLS_CA:+--ca "$SB_TLS_CA"}) || die "SB_EMAIL_SERVER is refused (above)"
  email="$SB_EMAIL_SERVER from $SB_EMAIL_FROM${SB_EMAIL_LOGIN_FILE:+, signing in with $SB_EMAIL_LOGIN_FILE}"
fi
if [ "${SB_DATABASE:-bundled}" = external ]; then
  [ -f "${SB_DATABASE_DSN_FILE:-}" ] || die "SB_DATABASE=external needs SB_DATABASE_DSN_FILE"
  d=$(bash tools/selfhost/sb.sh selfhost check-dsn "$SB_DATABASE_DSN_FILE") || die "SB_DATABASE_DSN_FILE is refused (above)"
  database="external, ${d#database: }"
else
  database="bundled (the postgres container on this machine)"
fi
if [ -n "$CHECK" ]; then
  echo "install: outbound proxy: $proxy"
  echo "install: database: $database"
  echo "install: email: $email"
  echo "install: off-box copies: $offsite"
  echo "install: the machine and $CONFIG look right; nothing changed"; exit 0
fi
if [ -n "${SB_SHARE:-}" ]; then # stewards' notes (SB-1705): the share's notes/ folder, the only one api writes
  SB_NOTES=${SB_NOTES:-$SB_SHARE/notes}; mkdir -p "$SB_NOTES"
  chown 65532:65532 "$SB_NOTES" 2>/dev/null || echo "install: couldn't give $SB_NOTES to uid 65532; stewards can't publish notes" >&2
  export SB_NOTES
fi
[[ "$SB_OFFSITE" == s3://* ]] || chown 65532:65532 "$SB_OFFSITE" 2>/dev/null || echo "install: couldn't give $SB_OFFSITE to uid 65532; the backup job must be able to write there" >&2

# The images: the bundle's, or built here.
if ls images/*.tar >/dev/null 2>&1; then
  for f in images/*.tar; do docker load -q -i "$f" >/dev/null; done
  echo "install: loaded the bundle's images"
elif command -v go >/dev/null; then
  bash tools/selfhost/images.sh >/dev/null
else
  die "no images: run this from an install bundle (make selfhost-bundle), or install Go to build them"
fi

SB_CONFIG=$CONFIG SKIP_IMAGES=1 bash tools/selfhost/selfhost.sh up
bash tools/selfhost/update.sh

# systemd: the cell at boot, upgrades hourly, the database dumped nightly, a restore drill monthly.
for u in deploy/selfhost/systemd/*.service deploy/selfhost/systemd/*.timer; do
  sed "s|/opt/second-brain|$HOME_DIR|g" "$u" >"/etc/systemd/system/$(basename "$u")"
done
systemctl daemon-reload
systemctl enable --now sb-selfhost.service sb-selfhost-update.timer sb-selfhost-dbdump.timer sb-selfhost-drill.timer >/dev/null

echo "install: $SB_TENANT is running at https://$SB_HOST$([ "${SB_PORT:-443}" = 443 ] || echo ":$SB_PORT")"
if [ "${SB_IDP:-bundled}" = bundled ]; then
  echo "install: add people with:  printf '%s' PASSWORD | $HOME_DIR/tools/selfhost/compose.sh run --rm -T idp user add --data /var/lib/sb-idp \\"
  echo "           --email ann@$SB_HOST --scim http://api:8080/scim/v2 --scim-token-file /run/sb/scim"
fi
[ ! -f build/selfhost/recovery-key.private ] || echo "install: now take the recovery key off the box: $HOME_DIR/tools/selfhost/install.sh --recovery-kit --custodian 1=NAME … (sb-selfhost(7), The recovery kit)"
echo "install: then run the laws inside it: SB_TOKEN=\$(cat build/selfhost/vault/api/vault/tenants/$SB_TENANT/control-plane) sb tenant verify $SB_MANIFEST --cell https://$SB_HOST"
