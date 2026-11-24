---
title: "SB-3603: Review every N days"
summary: "Every document in a space comes due on the same schedule."
---

# SB-3603: Review every N days

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.1 |
| Stories | US-36.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-13 |
| Depends on | SB-2501 |
| Status | Done |

## Why

Every document in a space comes due on the same schedule.

## Acceptance criteria

- [x] A review mark takes `days` of 30, 90, 180 or 365, and the document comes due at its end
- [x] Owners marking their own documents get the same choice
- [x] Without `days` nothing changes; another value gives EINVAL

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Marks
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

A mark takes its own review interval of 30, 90, 180 or 365 days

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
