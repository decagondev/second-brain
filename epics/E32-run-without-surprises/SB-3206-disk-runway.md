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
| Status | Open |

## Why

Nobody sees the disk filling until it is full.

## Acceptance criteria

- [ ] The host step keeps 14 days of the data disk's used bytes; the doctor and the console estimate days until full, warn under 30 and alert under 7
- [ ] Fewer than 3 days of history shows not enough history, never a number
- [ ] Steady growth gives the right estimate; a shrinking disk shows no runway

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Host step
- [ ] Doctor and console
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
