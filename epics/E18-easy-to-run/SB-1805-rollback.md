---
title: "SB-1805: Roll back the box in one command"
summary: "When an upgrade misbehaves, the box has no rollback: IT would have to edit compose.env by hand."
---

# SB-1805: Roll back the box in one command

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.2 |
| Stories | US-18.2 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06, LAW-08 |
| Depends on | — |
| Status | Open |

## Why

When an upgrade misbehaves, the box has no rollback: IT would have to edit compose.env by hand.

## Acceptance criteria

- [ ] Each upgrade keeps the previous admitted digests; `update.sh --rollback` starts exactly those, records them as running, and holds the timer
- [ ] A rollback never starts digests the cell didn't admit, and never goes two releases back
- [ ] `update.sh --resume` lifts the hold; the console shows a held box

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Previous record
- [ ] Rollback and resume
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
