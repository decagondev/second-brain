---
title: "SB-1803: One-command restore and a monthly restore drill"
summary: "Restore is five commands in a runbook nobody has run since the install, and nothing proves last night's dump restores."
---

# SB-1803: One-command restore and a monthly restore drill

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.2 |
| Stories | US-18.2 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-11, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

Restore is five commands in a runbook nobody has run since the install, and nothing proves last night's dump restores.

## Acceptance criteria

- [x] `tools/selfhost/restore.sh [--dump FILE]` checks the dump is for this cell, stops the writers, restores, starts them and runs `sb audit verify`
- [x] A dump made by another cell's keys is refused before anything stops
- [x] `restore.sh --drill` restores the newest dump into a scratch database, compares row counts per table with the live one, drops it and records the result; `sb-selfhost-drill.timer` runs it monthly and a failure is an alert

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] restore.sh
- [x] Drill mode and timer
- [x] Runbook
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.2): `tools/selfhost/restore.sh` restores the newest dump (or `--dump FILE`): `dbdump.sh` now writes `NAME.dump.json` beside each dump (the tenant and a 16-hex fingerprint of the audit anchor seed), and a dump of another tenant or other keys is refused before anything stops; then it stops the writers, runs `pg_restore --clean`, starts them, waits for `/readyz` and runs `sb audit verify` against the cell. `restore.sh --drill` restores the newest dump into `sb_drill` inside the postgres container, checks every live table is there, drops it and writes `build/selfhost/drill.json`; `sb-selfhost-drill.timer` runs it monthly and the install enables it; the doctor and the console show the last drill (warn after 35 days, fail after 62 or on a failed drill). Run on the demo box here: the drill passed (6 tables), a dump whose record names other keys was refused with the cell left running, and a real restore came back ready with the chain verified. That run found BUG-0027 (`sb audit verify` on a cell with no rows yet).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
