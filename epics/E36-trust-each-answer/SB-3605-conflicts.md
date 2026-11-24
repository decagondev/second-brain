---
title: "SB-3605: Conflicts"
summary: "Answers mark when sources disagree, but nobody sees which documents keep disagreeing."
---

# SB-3605: Conflicts

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.2 |
| Stories | US-36.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-3304 |
| Status | Done |

## Why

Answers mark when sources disagree, but nobody sees which documents keep disagreeing.

## Acceptance criteria

- [x] The review list's Conflicts names pairs of documents answers marked as disagreeing this month, with a count, most first
- [x] Only pairs the steward can open both of are named; the rest are counted
- [x] Counts only; no asker or question

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Review list
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Conflicts on the review list: pairs of documents disagreeing claims cited this month, named only when the steward can open both

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
