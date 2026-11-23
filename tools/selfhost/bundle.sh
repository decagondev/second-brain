#!/usr/bin/env bash
# make selfhost-bundle (sb-selfhost(7), Install; SB-1605): everything an install needs on a
# machine with no Go, no Node and, if need be, no internet: every image the layout runs,
# saved by name, and the files install.sh and selfhost.sh read. The images are the ones
# make images built here, so their ids are the ones a release names.
#   bundle.sh        writes build/selfhost-bundle/second-brain-selfhost-COMMIT.tar.gz
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
OUT=build/selfhost-bundle
ROOT=$OUT/second-brain-selfhost-$COMMIT
[ "${SKIP_IMAGES:-}" = 1 ] || DOCKER="$DOCKER" bash tools/selfhost/images.sh >/dev/null
rm -rf "$ROOT" && mkdir -p "$ROOT/images"
pinned=$(grep -hoE '(caddy|pgvector/pgvector|versity/versitygw)@sha256:[0-9a-f]{64}' deploy/selfhost/compose.yaml | sort -u)
for img in $pinned; do "$DOCKER" image inspect "$img" >/dev/null 2>&1 || "$DOCKER" pull -q "$img" >/dev/null; done
i=0
for img in sb-api:local sb-ingest:local sb-jobs:local sb-updater:local sb-idp:local sb-wormsetup:local $pinned; do
  i=$((i + 1))
  f=$ROOT/images/$(printf '%02d' $i)-$(printf '%s' "${img%%@*}" | tr '/:' '--').tar
  if command -v wslpath >/dev/null && [[ "$DOCKER" == *.exe ]]; then "$DOCKER" save -o "$(wslpath -w "$f")" "$img"; else "$DOCKER" save -o "$f" "$img"; fi
done
mkdir -p "$ROOT/tools" "$ROOT/deploy" "$ROOT/examples" "$ROOT/policy" "$ROOT/build/images" "$ROOT/bin"
# The sb command for the host-side steps (doctor, bundle checks, sealing): static, no Go needed.
CGO_ENABLED=0 go build -trimpath -o "$ROOT/bin/sb" ./cmd/sb
cp -r tools/selfhost "$ROOT/tools/"
rm -rf "$ROOT/tools/selfhost/devrelease" "$ROOT/tools/selfhost/wormsetup"
cp -r deploy/selfhost "$ROOT/deploy/"
cp -r policy/platform "$ROOT/policy/"
cp -r examples/share examples/policies "$ROOT/examples/"
cp build/images/digests.txt build/images/*.spdx.json "$ROOT/build/images/" 2>/dev/null || true
cp manual/sb-selfhost.7.md manual/sb-idp.1.md "$ROOT/"
# The model profile's weights and its pinned image (SB-1710), when the operator names them,
# so an air-gapped site never downloads them; install runs model.sh with SB_MODEL_GGUF and
# SB_EMBED_GGUF pointing here.
if [ -n "${SB_MODEL_GGUF:-}" ] && [ -n "${SB_EMBED_GGUF:-}" ]; then
  mkdir -p "$ROOT/models"
  cp "$SB_MODEL_GGUF" "$ROOT/models/model.gguf" && cp "$SB_EMBED_GGUF" "$ROOT/models/embed.gguf"
  (cd "$ROOT/models" && sha256sum ./*.gguf >SHA256SUMS)
  if [ -f build/selfhost/model.json ]; then
    img=$(sed -nE 's/^ *"image": *"([^"]+)".*/\1/p' build/selfhost/model.json)
    f=$ROOT/images/99-model-server.tar
    if command -v wslpath >/dev/null && [[ "$DOCKER" == *.exe ]]; then "$DOCKER" save -o "$(wslpath -w "$f")" "$img"; else "$DOCKER" save -o "$f" "$img"; fi
  fi
fi
(cd "$ROOT/images" && sha256sum ./*.tar >SHA256SUMS)
tar -C "$OUT" -czf "$OUT/second-brain-selfhost-$COMMIT.tar.gz" "second-brain-selfhost-$COMMIT"
rm -rf "$ROOT"
echo "bundle: $OUT/second-brain-selfhost-$COMMIT.tar.gz"
