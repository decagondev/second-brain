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
| Status | Done |

## Why

Cites to a PDF or a deck stop at the document, not the page or slide.

## Acceptance criteria

- [x] Ingest records the PDF page and slide number for each chunk (`Chunk.Page`)
- [x] Cites name them and the passage view opens there
- [x] Older chunks without a page still cite by line

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Page and slide at ingest
- [x] Cite names
- [x] Passage view
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `ports.Document.Pages` (where each page starts in the text, in runes); the folder connector's `Pages` for `.pdf` (each content stream, `pdfPages`) and `.pptx` (each slide, `fromZipParts`). `ingest.PageAt` sets `Chunk.Page`, kept in Postgres (`page`), carried to `answer.Cite.page` and `Passage.page`; the web's `citeName` shows "Title, slide 7" for a deck and "Title, page 12" otherwise. A chunk without pages cites as before. The passage view shows the page with the passage rather than opening the file at it: the cell serves passages, not files. Tested in TestPagesOfPDFsAndDecks.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
