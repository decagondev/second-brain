---
title: "SB-1807: Use the company's Postgres: two VMs"
summary: "Some sites run Postgres on its own server with its own backups and want the cell on another VM."
---

# SB-1807: Use the company's Postgres: two VMs

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.3 |
| Stories | US-18.3 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-02 |
| Depends on | — |
| Status | Done |

## Why

Some sites run Postgres on its own server with its own backups and want the cell on another VM.

## Acceptance criteria

- [x] `SB_DATABASE=external` with `SB_DATABASE_DSN_FILE`: the layout doesn't start `postgres`, and the services use that server
- [x] A DSN without `sslmode=verify-full` is refused (loopback excepted)
- [x] `dbdump.sh` and `restore.sh` work against it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] install.env, selfhost.sh, compose
- [x] DSN check in Go with tests
- [x] Docs
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.3): `SB_DATABASE=external` with `SB_DATABASE_DSN_FILE` leaves `postgres` out (it is in compose profile `db`, which `compose.sh` adds unless `SB_DATABASE=external`, and `depends_on` has `required: false`); the string goes into the `api`, `ingest` and `jobs` vaults, and the bundled string is written back every run, so moving back works. `box.CheckDSN` (`sb selfhost check-dsn`) refuses anything but `sslmode=verify-full` off loopback, in both the URL and key=value forms, a Unix socket, or another scheme, and never repeats the password; `SB_DATABASE_CA` goes to `/etc/sb/ca`. `dbdump.sh` and `restore.sh` share `tools/selfhost/pg.sh`: against an external database the client runs in a throwaway container with a `pg_service.conf` (0600, removed after; `check-dsn --service`), so the password is in no process's arguments or environment. The drill now restores into a container with no network for both layouts and only reads the live database. Run here against a second Postgres with its own CA and `ssl=on`: the cell connected over TLS (`pg_stat_ssl`), the dump, the drill and a restore with `sb audit verify` passed, and moving back to bundled restored the old data. Tests: `TestCheckDSN`, `TestDSNParams`, `TestSelfhostCheckDSNService`, `TestLAW02_ExternalDatabaseIsVerified` with a self-test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
