---
title: "SB-2501: Reviewed, still right"
summary: "A stale document that is still right stays on the review list for good."
---

# SB-2501: Reviewed, still right

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.1 |
| Stories | US-25.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-13, LAW-11 |
| Depends on | — |
| Status | Done |

## Why

A stale document that is still right stays on the review list for good.

## Acceptance criteria

- [x] Reviewed marks a document at its current version, with an audit row; it leaves the list until a new version or `review_after_days` again
- [x] A wrong-citation flag after the mark brings it back at once
- [x] Mark, rebuild, still off; a new version, back on

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API and record
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S24.1; see manual/sb-ask.3.md

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
