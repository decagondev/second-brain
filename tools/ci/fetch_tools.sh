#!/usr/bin/env sh
# Fetch pinned tool jars into tools/vendor/ and refuse any whose checksum doesn't match.
set -eu
. "$(dirname "$0")/../versions.env"
mkdir -p tools/vendor
fetch() { # url file sha
  [ -f "tools/vendor/$2" ] || curl -fsSL -o "tools/vendor/$2" "$1"
  echo "$3  tools/vendor/$2" | sha256sum -c - || { echo "checksum mismatch for $2" >&2; rm -f "tools/vendor/$2"; exit 1; }
}
fetch "https://github.com/tlaplus/tlaplus/releases/download/v${TLA2TOOLS_VERSION}/tla2tools.jar" tla2tools.jar "$TLA2TOOLS_SHA256"
fetch "https://github.com/AlloyTools/org.alloytools.alloy/releases/download/v${ALLOY_VERSION}/org.alloytools.alloy.dist.jar" alloy.jar "$ALLOY_SHA256"
