---
title: "SB-2507: Stewards' line in the digest"
summary: "Stewards have no weekly view of their queue."
---

# SB-2507: Stewards' line in the digest

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.2 |
| Stories | US-25.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-21 |
| Depends on | — |
| Status | Done |

## Why

Stewards have no weekly view of their queue.

## Acceptance criteria

- [x] A steward's weekly digest adds one line per space they steward: answered, still open, the oldest's age; counts only
- [x] Only spaces they steward when the digest is sent
- [x] Someone removed as a steward gets no line for that space

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Digest line
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S24.2; see manual/sb-ask.3.md

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
