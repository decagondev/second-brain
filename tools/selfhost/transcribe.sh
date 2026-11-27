#!/usr/bin/env bash
# make selfhost-transcribe (sb-selfhost(7), Recordings on the box; SB-6102): recordings in
# the share transcribed by whisper.cpp's server on this machine, with no sound leaving it.
#
#   SB_WHISPER_MODEL=PATH   the whisper model's weights (ggml .bin), fetched by the operator
#   SB_TRANSCRIBE_SAMPLE=PATH  a short recording (WAV) of a person speaking
#   SB_TRANSCRIBE_EXPECT="words"  what is said in it, every word of which must be heard
#   SB_WHISPER_IMAGE_REF    the server image to pin (default ghcr.io/ggml-org/whisper.cpp:main)
#   DOCKER, CURL            the Docker command and curl (curl.exe on Docker Desktop from WSL)
#
# It pins the image by digest, records the weights' SHA-256, copies the weights into the
# models volume, transcribes the sample through a loopback-only port removed afterwards and
# checks every expected word was heard. Only if it was does it put the sample in the cell's
# state (so the evals job gates the tier for each release, LAW-06), write
# build/selfhost/transcribe.json and run selfhost.sh, which adds the transcribe profile.
set -euo pipefail
cd "$(dirname "$0")/../.."
DOCKER=${DOCKER:-docker}
CURL=${CURL:-curl}
die() { echo "selfhost-transcribe: $*" >&2; exit 1; }
: "${SB_WHISPER_MODEL:?set SB_WHISPER_MODEL to the weights of the whisper model}"
: "${SB_TRANSCRIBE_SAMPLE:?set SB_TRANSCRIBE_SAMPLE to a short WAV of someone speaking}"
: "${SB_TRANSCRIBE_EXPECT:?set SB_TRANSCRIBE_EXPECT to the words said in it}"
[ -f "$SB_WHISPER_MODEL" ] && [ -f "$SB_TRANSCRIBE_SAMPLE" ] || die "a file isn't there"
REF=${SB_WHISPER_IMAGE_REF:-ghcr.io/ggml-org/whisper.cpp:main}
HELPER=caddy@sha256:226d1f059b75399fe19182893c7184591c07b97afc8dfcf44eeb80c9a77a530f
DIR=build/selfhost
mkdir -p "$DIR"
hostpath() { if command -v wslpath >/dev/null 2>&1 && [[ "$DOCKER" == *.exe ]]; then wslpath -w "$1"; else echo "$1"; fi; }

case "$REF" in
*@sha256:*) PINNED=$REF ;;
*) "$DOCKER" pull "$REF" >/dev/null
   PINNED=$("$DOCKER" image inspect -f '{{index .RepoDigests 0}}' "$REF" | tr -d '\r')
   [[ "$PINNED" == *@sha256:* ]] || die "couldn't resolve $REF to a digest" ;;
esac
echo "selfhost-transcribe: image $PINNED"

W=$(sha256sum "$SB_WHISPER_MODEL" | cut -d' ' -f1)
"$DOCKER" volume create --label com.docker.compose.project=sb-selfhost --label com.docker.compose.volume=models sb-selfhost_models >/dev/null
"$DOCKER" run --rm -u 0 -v sb-selfhost_models:/dst -v "$(hostpath "$(cd "$(dirname "$SB_WHISPER_MODEL")" && pwd)"):/m:ro" "$HELPER" \
  sh -c "cp '/m/$(basename "$SB_WHISPER_MODEL")' /dst/whisper.bin && chown 65532:65532 /dst/whisper.bin && chmod 0444 /dst/whisper.bin" >/dev/null

"$DOCKER" rm -f sb-transcribe-gate >/dev/null 2>&1 || true
"$DOCKER" run -d --rm --name sb-transcribe-gate -u 65532:65532 --read-only -p 127.0.0.1:18090:8080 -v sb-selfhost_models:/models:ro \
  --entrypoint /app/build/bin/whisper-server "$PINNED" --host 0.0.0.0 --port 8080 -m /models/whisper.bin >/dev/null
trap '"$DOCKER" rm -f sb-transcribe-gate >/dev/null 2>&1 || true' EXIT
for _ in $(seq 60); do "$CURL" -s http://127.0.0.1:18090/ >/dev/null 2>&1 && break; sleep 2; done
HEARD=$("$CURL" -sf -F "file=@$SB_TRANSCRIBE_SAMPLE" -F response_format=verbose_json http://127.0.0.1:18090/inference | tr '[:upper:]' '[:lower:]') ||
  die "the server didn't transcribe the sample"
for w in $(echo "$SB_TRANSCRIBE_EXPECT" | tr '[:upper:]' '[:lower:]' | tr -d '.,;:!?'); do
  case "$HEARD" in *"$w"*) ;; *) die "the sample wasn't heard right (missing \"$w\"); the cell keeps no transcriber" ;; esac
done
"$DOCKER" rm -f sb-transcribe-gate >/dev/null
trap - EXIT

# The sample and its words in the cell's state, for the evals job's gate each release.
"$DOCKER" volume create --label com.docker.compose.project=sb-selfhost --label com.docker.compose.volume=state sb-selfhost_state >/dev/null
printf '%s\n' "$SB_TRANSCRIBE_EXPECT" >"$DIR/sample.txt"
"$DOCKER" run --rm -u 0 -v sb-selfhost_state:/dst -v "$(hostpath "$(cd "$(dirname "$SB_TRANSCRIBE_SAMPLE")" && pwd)"):/s:ro" -v "$(hostpath "$(pwd)/$DIR"):/t:ro" "$HELPER" \
  sh -c "mkdir -p /dst/evals/transcribe && cp '/s/$(basename "$SB_TRANSCRIBE_SAMPLE")' /dst/evals/transcribe/sample.wav && cp /t/sample.txt /dst/evals/transcribe/sample.txt && chown -R 65532:65532 /dst/evals && chmod 700 /dst/evals/transcribe" >/dev/null
rm -f "$DIR/sample.txt"

cat >"$DIR/transcribe.json" <<JSON
{
  "weights_sha256": "$W",
  "image": "$PINNED"
}
JSON
echo "selfhost-transcribe: the sample was heard; adding the transcribe profile (weights sha256:$W)"
exec bash tools/selfhost/selfhost.sh up
