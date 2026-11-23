# shellcheck shell=bash
# Sourced by dbdump.sh and restore.sh (sb-selfhost(7), Backup and restore; SB-1807): run a
# Postgres client against the cell's database. The bundled database is reached in its own
# container; with SB_DATABASE=external, a throwaway container of the same image runs the
# client on the egress network. The connection string, from the jobs service's vault, is
# turned by `sb selfhost check-dsn --service` into a pg_service.conf (0600, removed after)
# mounted read-only, so the password is never in a process's arguments or environment.
# Callers set DOCKER, DIR and ENV.
PGIMAGE=pgvector/pgvector@sha256:b513c48566e4777cd3370451400000c78eda6a08131d68f4c31d4a76f27bd8fd
pg_external() { grep -qx 'SB_DATABASE=external' "$ENV" 2>/dev/null; }
pg_hostpath() {
  if command -v wslpath >/dev/null && [[ "$DOCKER" == *.exe ]]; then wslpath -w "$1"; else realpath "$1"; fi
}
# pg TOOL [ARGS…]: TOOL is pg_dump, pg_restore or psql, against the cell's database; stdin
# and stdout pass through.
pg() {
  local tool=$1; shift
  if ! pg_external; then
    DOCKER="$DOCKER" bash tools/selfhost/compose.sh exec -T postgres "$tool" -U sb -d sb "$@"
    return
  fi
  local tenant root=system svc rc=0
  tenant=$(sed -nE 's/^SB_TENANT=//p' "$ENV" | tr -d '\r')
  [ -f "$DIR/ca/database-ca.crt" ] && root=/etc/sb/ca/database-ca.crt
  svc=$(mktemp "$DIR/.pg_service.XXXXXX")
  bash tools/selfhost/sb.sh selfhost check-dsn "$DIR/vault/jobs/vault/tenants/$tenant/database" --service "$svc" >/dev/null || { rm -f "$svc"; return 1; }
  "$DOCKER" run --rm -i --network sb-selfhost_egress -e PGSSLROOTCERT=$root -e PGSERVICEFILE=/run/sb/pg_service.conf \
    -v "$(pg_hostpath "$svc"):/run/sb/pg_service.conf:ro" -v "$(pg_hostpath "$DIR/ca"):/etc/sb/ca:ro" "$PGIMAGE" "$tool" -d service=sb "$@" || rc=$?
  rm -f "$svc"
  return $rc
}
# pg_scratch_start NAME / pg_scratch TOOL ARGS… / pg_scratch_stop: an empty database in a
# container with no network at all, for the restore drill.
pg_scratch_start() {
  PG_SCRATCH=$1
  "$DOCKER" rm -f "$PG_SCRATCH" >/dev/null 2>&1 || true
  "$DOCKER" run -d --rm --name "$PG_SCRATCH" --network none -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_DB=sb_drill "$PGIMAGE" >/dev/null
  for _ in $(seq 60); do
    "$DOCKER" exec "$PG_SCRATCH" pg_isready -h 127.0.0.1 -U postgres -d sb_drill -q 2>/dev/null && return 0
    sleep 1
  done
  return 1
}
pg_scratch() { local tool=$1; shift; "$DOCKER" exec -i "$PG_SCRATCH" "$tool" -h 127.0.0.1 -U postgres -d sb_drill "$@"; }
pg_scratch_stop() { [ -z "${PG_SCRATCH:-}" ] || "$DOCKER" rm -f "$PG_SCRATCH" >/dev/null 2>&1 || true; }
