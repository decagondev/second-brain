#!/usr/bin/env bash
# make images (SB-1601): build every cell service as a static binary and put each in its
# own image. Writes build/images/digests.txt (NAME sha256:…) and an SBOM per binary.
#   DOCKER   the Docker command (default docker; Docker Desktop's docker.exe on Windows)
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
OUT=build/images
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
CREATED=$(git show -s --format=%cI HEAD 2>/dev/null || echo 1970-01-01T00:00:00Z)
EPOCH=$(git show -s --format=%ct HEAD 2>/dev/null || echo 0) # the image's dates: the commit's, so a rebuild is the same image
rm -rf "$OUT/ctx" && mkdir -p "$OUT/ctx/bin" "$OUT/ctx/web"
flags=(-trimpath -buildvcs=false -ldflags "-s -w -buildid= -X example.com/second-brain/internal/version.Commit=$COMMIT")
for svc in api ingest jobs updater; do
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build "${flags[@]}" -o "$OUT/ctx/bin/$svc" "./cmd/sb-$svc"
done
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build "${flags[@]}" -o "$OUT/ctx/bin/devidp" ./tools/devidp
if [ -d web/dist ]; then cp -r web/dist/. "$OUT/ctx/web/"; else echo "images: web/dist not built (make web); the api image serves no UI" >&2; fi
cp deploy/images/Dockerfile "$OUT/ctx/Dockerfile"
find "$OUT/ctx" -exec touch -h -d "@$EPOCH" {} +
: >"$OUT/digests.txt"
for svc in api ingest jobs updater devidp; do
  "$DOCKER" buildx build -q --build-arg "SVC=$svc" --build-arg "SOURCE_DATE_EPOCH=$EPOCH" --provenance=false     --output "type=docker,name=sb-$svc:$COMMIT" "$OUT/ctx" >/dev/null
  "$DOCKER" tag "sb-$svc:$COMMIT" "sb-$svc:local"
  id=$("$DOCKER" image inspect --format '{{.Id}}' "sb-$svc:$COMMIT" | tr -d '\r')
  echo "$svc $id" >>"$OUT/digests.txt"
  [ "$svc" = devidp ] || go run ./tools/release/sbom --binary "$OUT/ctx/bin/$svc" --name "sb-$svc" --created "$CREATED" --out "$OUT/$svc.spdx.json"
done
cat "$OUT/digests.txt"
