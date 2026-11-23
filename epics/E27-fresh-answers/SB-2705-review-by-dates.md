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
| Status | Open |

## Why

A document that says when it should be reviewed is treated like any other.

## Acceptance criteria

- [ ] Ingest reads `review-by:` or `expires:` from Markdown front matter, or a custom `review-by` property in an Office file, onto the chunk
- [ ] A cite is stale from that date or from `review_after_days`, whichever comes first; a date never makes a document fresher
- [ ] A past date makes a new document stale; a date in 2099 changes nothing; a malformed date is ignored and counted

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ingest
- [ ] Stale rule
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
