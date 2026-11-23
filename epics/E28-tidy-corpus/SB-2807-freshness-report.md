---
title: "SB-2807: Freshness in the report"
summary: "Nobody sees whether answers are getting staler."
---

# SB-2807: Freshness in the report

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.2 |
| Stories | US-28.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | SB-2701 |
| Status | Done |

## Why

Nobody sees whether answers are getting staler.

## Acceptance criteria

- [x] The steward screen shows per space the share of the month's cited answers resting mostly on stale cites, and last month's
- [x] The weekly report adds one line for the cell; counts only
- [x] The share matches the traces; the line carries no title or question

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Counts
- [x] UI and report
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S27.2; tested in internal/core/ask/owned_test.go, internal/core/notify/owned_test.go, internal/core/report and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
