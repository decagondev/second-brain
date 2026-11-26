---
title: "SB-5302: Try a trigger first"
summary: "A new trigger goes live without anyone seeing what it would have done."
---

# SB-5302: Try a trigger first

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.1 |
| Stories | US-53.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-5301, SB-4704 |
| Status | Done |

## Why

A new trigger goes live without anyone seeing what it would have done.

## Acceptance criteria

- [x] **Try it** replays the last 14 days of matching events as the approver and shows what it would have drafted
- [x] It shows what each run would have cost, and nothing is sent
- [x] Events the approver can't open are counted, not shown

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Replay
- [x] Cost estimate
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `Stewards.Try` at `POST /v1/spaces/NAME/triggers/ON/try`: the space's work events of the last 14 days (`TryDays`), checked as the steward; at most 5 (`MaxTry`) drafted as probes (no conversation, nothing in a queue, nothing sent), the rest the steward can't open counted in `hidden`, and `cost_usd` from a tally of the probes. Tested in TestBorrowAccessThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
