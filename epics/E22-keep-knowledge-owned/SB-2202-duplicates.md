---
title: "SB-2202: Duplicate documents"
summary: "Copies of the same document in two places are both cited."
---

# SB-2202: Duplicate documents

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.1 |
| Stories | US-22.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-13, LAW-01 |
| Depends on | — |
| Status | Open |

## Why

Copies of the same document in two places are both cited.

## Acceptance criteria

- [ ] Ingest keeps a hash of each document's normalised text, rebuilt with the index
- [ ] Documents with the same hash in a space are listed once on the review list, with the locations the steward can open
- [ ] An answer that would cite two copies cites the newest the asker can open; delete it, and it cites the older

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Text hash in ingest
- [ ] Dedup in answers
- [ ] Review list kind
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
