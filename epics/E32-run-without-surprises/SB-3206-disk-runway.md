---
title: "SB-3206: Disk runway"
summary: "Nobody sees the disk filling until it is full."
---

# SB-3206: Disk runway

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.2 |
| Stories | US-32.2 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-03 |
| Depends on | — |
| Status | Done |

## Why

Nobody sees the disk filling until it is full.

## Acceptance criteria

- [x] The host step keeps 14 days of the data disk's used bytes; the doctor and the console estimate days until full, warn under 30 and alert under 7
- [x] Fewer than 3 days of history shows not enough history, never a number
- [x] Steady growth gives the right estimate; a shrinking disk shows no runway

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Host step
- [x] Doctor and console
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

box.Runway fits the used-bytes trend from state/box/disk.json (at most one sample a day, kept by wire.withDiskHistory, read by facts.sh); the disk runway line passes with not enough history yet or not growing, warns under 30 days and fails under 7, and its fix names docs/runbooks/selfhost.md#disk-runway. Test: box/runway_test.go.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
