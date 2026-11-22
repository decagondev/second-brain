#!/usr/bin/env bash
# make selfhost-model (sb-selfhost(7), A model on the box; SB-1710): answers and embeddings
# from models on this machine, with no question leaving it.
#
#   SB_MODEL_GGUF=PATH  the instruct model's weights (GGUF), fetched by the operator
#   SB_EMBED_GGUF=PATH  the embedding model's weights (GGUF)
#   SB_EMBED_DIM=N      the embedding model's vector size (e.g. 768)
#   SB_MODEL_ID, SB_EMBED_ID  model ids (default: the file names)
#   SB_MODEL_IMAGE_REF  llama.cpp's server image to pin (default ghcr.io/ggml-org/llama.cpp:server)
#   DOCKER, CURL        the Docker command and curl (curl.exe on Docker Desktop from WSL)
#
# It pins the image by digest, records the weights' SHA-256, copies the weights into the
# models volume, runs the eval gate on that exact model through a loopback-only port that is
# removed afterwards, and only if the gate passes writes build/selfhost/model.json and runs
# selfhost.sh, which switches the cell to the model profile. A model that fails the gate is
# never configured. The index is rebuilt for the new embedder by the ingest service.
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
die() { echo "selfhost-model: $*" >&2; exit 1; }
: "${SB_MODEL_GGUF:?set SB_MODEL_GGUF to the GGUF file of the instruct model}"
: "${SB_EMBED_GGUF:?set SB_EMBED_GGUF to the GGUF file of the embedding model}"
: "${SB_EMBED_DIM:?set SB_EMBED_DIM to the vector size of the embedding model}"
[ -f "$SB_MODEL_GGUF" ] && [ -f "$SB_EMBED_GGUF" ] || die "a weights file isn't there"
case "$SB_EMBED_DIM" in ''|*[!0-9]*) die "SB_EMBED_DIM must be a number" ;; esac
SB_MODEL_ID=${SB_MODEL_ID:-$(basename "$SB_MODEL_GGUF" .gguf)}
SB_EMBED_ID=${SB_EMBED_ID:-$(basename "$SB_EMBED_GGUF" .gguf)}
REF=${SB_MODEL_IMAGE_REF:-ghcr.io/ggml-org/llama.cpp:server}
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f  # has a shell, as in selfhost.sh
DIR=build/selfhost
mkdir -p "$DIR"
hostpath() { if command -v wslpath >/dev/null 2>&1 && [[ "$DOCKER" == *.exe ]]; then wslpath -w "$1"; else echo "$1"; fi; }

# 1. Pin the server image by digest: what runs is what was gated.
case "$REF" in
*@sha256:*) PINNED=$REF ;;
*) "$DOCKER" pull "$REF" >/dev/null
   PINNED=$("$DOCKER" image inspect -f '{{index .RepoDigests 0}}' "$REF" | tr -d '\r')
   [[ "$PINNED" == *@sha256:* ]] || die "couldn't resolve $REF to a digest" ;;
esac
echo "selfhost-model: image $PINNED"

# 2. The weights' hashes, then the weights into the models volume, read-only for the servers.
W=$(sha256sum "$SB_MODEL_GGUF" | cut -d' ' -f1)
E=$(sha256sum "$SB_EMBED_GGUF" | cut -d' ' -f1)
"$DOCKER" volume create --label com.docker.compose.project=sb-selfhost --label com.docker.compose.volume=models sb-selfhost_models >/dev/null
"$DOCKER" run --rm -u 0 -v sb-selfhost_models:/dst -v "$(hostpath "$(cd "$(dirname "$SB_MODEL_GGUF")" && pwd)"):/m:ro" \
  -v "$(hostpath "$(cd "$(dirname "$SB_EMBED_GGUF")" && pwd)"):/e:ro" "$HELPER" \
  sh -c "cp '/m/$(basename "$SB_MODEL_GGUF")' /dst/model.gguf && cp '/e/$(basename "$SB_EMBED_GGUF")' /dst/embed.gguf && chown -R 65532:65532 /dst && chmod 0444 /dst/*.gguf" >/dev/null

# 3. The eval gate on that exact model, through a loopback port that exists only for the run.
"$DOCKER" rm -f sb-model-gate >/dev/null 2>&1 || true
"$DOCKER" run -d --rm --name sb-model-gate -u 65532:65532 --read-only -p 127.0.0.1:18080:8080 -v sb-selfhost_models:/models:ro \
  "$PINNED" --host 0.0.0.0 --port 8080 -m /models/model.gguf --ctx-size 8192 --no-webui >/dev/null
trap '"$DOCKER" rm -f sb-model-gate >/dev/null 2>&1 || true' EXIT
for _ in $(seq 120); do "$CURL" -sf http://127.0.0.1:18080/health >/dev/null 2>&1 && break; sleep 5; done
"$CURL" -sf http://127.0.0.1:18080/health >/dev/null 2>&1 || die "the model server didn't come up"
rm -rf build/evals-model
SUITES="evals/suites/permission.yaml evals/suites/citation.yaml evals/suites/injection.yaml evals/suites/refusal.yaml evals/suites/regression.yaml evals/suites/space-leak.yaml evals/suites/space-redteam.yaml evals/suites/agent-*.yaml evals/suites/template-*.yaml"
# shellcheck disable=SC2086
python3 -m evals run $SUITES --model "openaicompat=http://127.0.0.1:18080/v1#$SB_MODEL_ID" --results build/evals-model --release rel_model ||
  die "the model failed an eval suite; the cell keeps its model"
python3 -m evals gate build/evals-model/rel_model || die "the model failed the eval gate; the cell keeps its model"
"$DOCKER" rm -f sb-model-gate >/dev/null
trap - EXIT

# 4. Only now: the model's pins, for the release bundle and selfhost.sh, and the switch.
cat >"$DIR/model.json" <<JSON
{
  "model": "$SB_MODEL_ID",
  "weights_sha256": "$W",
  "embedder": "$SB_EMBED_ID@sha256:$E",
  "embed_model": "$SB_EMBED_ID",
  "embed_dim": $SB_EMBED_DIM,
  "image": "$PINNED"
}
JSON
echo "selfhost-model: the gate passed; switching the cell to $SB_MODEL_ID (weights sha256:$W)"
exec bash tools/selfhost/selfhost.sh up
