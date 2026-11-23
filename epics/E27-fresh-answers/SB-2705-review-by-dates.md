---
title: "SB-2705: Review-by dates in the document"
summary: "A document that says when it should be reviewed is treated like any other."
---

# SB-2705: Review-by dates in the document

| Field | Value |
|---|---|
| Epic | [E27 Fresh Answers](README.md) |
| Wave / slice | W26 / S26.2 |
| Stories | US-27.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-09, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

A document that says when it should be reviewed is treated like any other.

## Acceptance criteria

- [x] Ingest reads `review-by:` or `expires:` from Markdown front matter, or a custom `review-by` property in an Office file, onto the chunk
- [x] A cite is stale from that date or from `review_after_days`, whichever comes first; a date never makes a document fresher
- [x] A past date makes a new document stale; a date in 2099 changes nothing; a malformed date is ignored and counted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ingest
- [x] Stale rule
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S26.2; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
