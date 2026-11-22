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
| Status | Open |

## Why

Restore is five commands in a runbook nobody has run since the install, and nothing proves last night's dump restores.

## Acceptance criteria

- [ ] `tools/selfhost/restore.sh [--dump FILE]` checks the dump is for this cell, stops the writers, restores, starts them and runs `sb audit verify`
- [ ] A dump made by another cell's keys is refused before anything stops
- [ ] `restore.sh --drill` restores the newest dump into a scratch database, compares row counts per table with the live one, drops it and records the result; `sb-selfhost-drill.timer` runs it monthly and a failure is an alert

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] restore.sh
- [ ] Drill mode and timer
- [ ] Runbook
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
