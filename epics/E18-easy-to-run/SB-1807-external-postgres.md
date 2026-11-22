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
| Status | Open |

## Why

Some sites run Postgres on its own server with its own backups and want the cell on another VM.

## Acceptance criteria

- [ ] `SB_DATABASE=external` with `SB_DATABASE_DSN_FILE`: the layout doesn't start `postgres`, and the services use that server
- [ ] A DSN without `sslmode=verify-full` is refused (loopback excepted)
- [ ] `dbdump.sh` and `restore.sh` work against it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] install.env, selfhost.sh, compose
- [ ] DSN check in Go with tests
- [ ] Docs
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
