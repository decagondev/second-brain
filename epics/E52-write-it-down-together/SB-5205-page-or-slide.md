---
title: "SB-5205: Cite the page or slide"
summary: "Cites to a PDF or a deck stop at the document, not the page or slide."
---

# SB-5205: Cite the page or slide

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.2 |
| Stories | US-52.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-14 |
| Depends on | SB-3001, SB-3006, SB-2003 |
| Status | Open |

## Why

Cites to a PDF or a deck stop at the document, not the page or slide.

## Acceptance criteria

- [ ] Ingest records the PDF page and slide number for each chunk (`Chunk.Page`)
- [ ] Cites name them and the passage view opens there
- [ ] Older chunks without a page still cite by line

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Page and slide at ingest
- [ ] Cite names
- [ ] Passage view
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
