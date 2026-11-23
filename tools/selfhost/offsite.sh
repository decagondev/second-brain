# shellcheck shell=bash
# Sourced by dbdump.sh, kit.sh, restore.sh, retire.sh and facts.sh (sb-selfhost(7),
# Off-box copies to S3; SB-2303): the off-box copy is a mounted directory (SB_OFFSITE), or
# the company's bucket when compose.env has SB_OFFSITE_S3, reached with sb backup
# push|pull|list|check and the key pair in the jobs service's vault. The key pair is read by
# sb, never printed. Callers set DIR and ENV.
off_url() { sed -nE 's/^SB_OFFSITE_S3=//p' "$ENV" 2>/dev/null | tr -d '\r' | tail -1; }
off_keys() { printf '%s/vault/jobs/vault/tenants/%s/offsite' "$DIR" "$(sed -nE 's/^SB_TENANT=//p' "$ENV" | tr -d '\r' | tail -1)"; }
# off_put FILE NAME: FILE into the bucket as NAME, written once; sb refuses a file that is
# neither sealed nor a record. OFF_KEYS_JSON, when set, is the key pair itself (retire.sh,
# after build/selfhost is gone), handed over on a pipe.
off_put() {
  if [ -n "${OFF_KEYS_JSON:-}" ]; then
    bash tools/selfhost/sb.sh -q backup push --to "$(off_url)" --keys <(printf '%s' "$OFF_KEYS_JSON") --as "$2" "$1"
  else
    bash tools/selfhost/sb.sh -q backup push --to "$(off_url)" --keys "$(off_keys)" --as "$2" "$1"
  fi
}
# off_get NAME... DIR: objects from the bucket into DIR, each by its base name.
off_get() {
  local out=${*: -1}
  bash tools/selfhost/sb.sh -q backup pull --from "$(off_url)" --keys "$(off_keys)" --out "$out" "${@:1:$#-1}"
}
