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
| Status | Done |

## Why

When an upgrade misbehaves, the box has no rollback: IT would have to edit compose.env by hand.

## Acceptance criteria

- [x] Each upgrade keeps the previous admitted digests; `update.sh --rollback` starts exactly those, records them as running, and holds the timer
- [x] A rollback never starts digests the cell didn't admit, and never goes two releases back
- [x] `update.sh --resume` lifts the hold; the console shows a held box

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Previous record
- [x] Rollback and resume
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.2): an upgrade that changes a digest first writes `build/selfhost/previous.env` (each service's image id, attested reference and release). `update.sh --rollback` checks each reference is still attested and each image still held, starts them, checks each container runs exactly its digest, records them as running, writes `build/selfhost/held` and forgets the record; the hourly run then says it is held and changes nothing, a second rollback is refused, and `--resume` lifts the hold. The doctor and the console show a held box as a warning. Run on the demo box here: upgrade, rollback (the old digests running), a held timer run, a refused second rollback, resume, and the next run back on the new release.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
