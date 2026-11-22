#!/usr/bin/env bash
# install.sh (sb-selfhost(7), Install; SB-1605): a self-hosted cell on one Linux VM or
# on-prem server, from a checkout or an install bundle (make selfhost-bundle).
#
#   sudo tools/selfhost/install.sh [--install-docker] [--check] CONFIG
#
#   --install-docker   install Docker Engine and the compose plugin from the distribution
#                      (Debian or Ubuntu) when they are missing; otherwise they must be there
#   --check            check the machine and CONFIG, change nothing
#
# It loads the bundle's images (or builds them, in a checkout with Go), runs selfhost.sh
# with CONFIG, applies the first release, installs the systemd units (the cell at boot,
# hourly upgrades, nightly database dumps), and says how to add the first user and run
# sb tenant verify. Every key and secret is made on this machine and stays on it.
set -euo pipefail
cd "$(dirname "$0")/../.."
HOME_DIR=$(pwd)
die() { echo "install: $*" >&2; exit 1; }
INSTALL_DOCKER= CHECK= CONFIG=
for a in "$@"; do
  case "$a" in
  --install-docker) INSTALL_DOCKER=1 ;;
  --check) CHECK=1 ;;
  -*) die "unknown option $a" ;;
  *) CONFIG=$a ;;
  esac
done
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
[ -d "${SB_OFFSITE:-/nonexistent}" ] || die "SB_OFFSITE ($SB_OFFSITE) must be a mounted directory on another machine"
if [ "$(findmnt -n -o TARGET -T "$SB_OFFSITE" 2>/dev/null || echo /)" = / ]; then
  echo "install: SB_OFFSITE ($SB_OFFSITE) is on this machine's root disk; the off-box copy must be on another machine" >&2
fi
if [ -n "$CHECK" ]; then echo "install: the machine and $CONFIG look right; nothing changed"; exit 0; fi
if [ -n "${SB_SHARE:-}" ]; then # stewards' notes (SB-1705): the share's notes/ folder, the only one api writes
  SB_NOTES=${SB_NOTES:-$SB_SHARE/notes}; mkdir -p "$SB_NOTES"
  chown 65532:65532 "$SB_NOTES" 2>/dev/null || echo "install: couldn't give $SB_NOTES to uid 65532; stewards can't publish notes" >&2
  export SB_NOTES
fi
chown 65532:65532 "$SB_OFFSITE" 2>/dev/null || echo "install: couldn't give $SB_OFFSITE to uid 65532; the backup job must be able to write there" >&2

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

# systemd: the cell at boot, upgrades hourly, the database dumped nightly.
for u in deploy/selfhost/systemd/*.service deploy/selfhost/systemd/*.timer; do
  sed "s|/opt/second-brain|$HOME_DIR|g" "$u" >"/etc/systemd/system/$(basename "$u")"
done
systemctl daemon-reload
systemctl enable --now sb-selfhost.service sb-selfhost-update.timer sb-selfhost-dbdump.timer >/dev/null

echo "install: $SB_TENANT is running at https://$SB_HOST$([ "${SB_PORT:-443}" = 443 ] || echo ":$SB_PORT")"
if [ "${SB_IDP:-bundled}" = bundled ]; then
  echo "install: add people with:  printf '%s' PASSWORD | $HOME_DIR/tools/selfhost/compose.sh run --rm -T idp user add --data /var/lib/sb-idp \\"
  echo "           --email ann@$SB_HOST --scim http://api:8080/scim/v2 --scim-token-file /run/sb/scim"
fi
echo "install: then run the laws inside it: SB_TOKEN=\$(cat build/selfhost/vault/api/vault/tenants/$SB_TENANT/control-plane) sb tenant verify $SB_MANIFEST --cell https://$SB_HOST"
