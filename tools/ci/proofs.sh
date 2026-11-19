#!/usr/bin/env sh
# SB-011..013, SB-705: run every formal model. Unknown output is a failure (LAW-03 spirit):
# a model passes only if the checker positively reports success.
set -eu
cd "$(dirname "$0")/../.."
root=$(pwd)
tla="$root/tools/vendor/tla2tools.jar"; alloy="$root/tools/vendor/alloy.jar"
out_dir=$(mktemp -d); trap 'rm -rf "$out_dir"' EXIT
fail=0
for cfg in proofs/tla/*.cfg; do
  spec="${cfg%.cfg}.tla"; name=$(basename "$spec" .tla)
  # TLC wants the bare file name (the module name must match), so run from the model's directory.
  out=$(cd "$(dirname "$spec")" && java -XX:+UseParallelGC -cp "$tla" tlc2.TLC -workers auto -deadlock \
        -metadir "$out_dir/$name" -config "$name.cfg" "$name.tla" 2>&1) || true
  if echo "$out" | grep -q "Model checking completed. No error has been found."; then
    echo "proofs: ok    $spec ($(echo "$out" | grep -o '[0-9,]* distinct states found' | head -1))"
  else
    echo "proofs: FAIL  $spec"; echo "$out" | tail -30; fail=1
  fi
done
# Alloy 6.2 CLI: `exec` runs only the default command, so run each `check` by name (no `*`:
# the Windows Java launcher expands it). A check passes only when Alloy prints UNSAT for it,
# and each `run` must be SAT, so the model is not vacuous.
for als in proofs/alloy/*.als; do
  for kind in check run; do
    for c in $(sed -n "s/^$kind \([A-Za-z0-9_]*\).*/\1/p" "$als"); do
      out=$(java -jar "$alloy" exec -f -c "$c" -t none -o "$out_dir/$c" "$als" 2>&1) || true
      want=UNSAT; [ "$kind" = run ] && want=SAT
      if echo "$out" | grep -Eq "^[0-9]+\. +$kind +$c +.* $want\$"; then
        echo "proofs: ok    $als $kind $c ($want)"
      else
        echo "proofs: FAIL  $als $kind $c (expected $want)"; echo "$out" | grep -v '^WARNING' | tail -30; fail=1
      fi
    done
  done
done
exit $fail
