---
title: "SB-2003: Word, OpenDocument and HTML on file shares"
summary: "The folder connector reads `.txt`, `.md` and `.pdf`; most company shares are mostly Word."
---

# SB-2003: Word, OpenDocument and HTML on file shares

| Field | Value |
|---|---|
| Epic | [E20 Close the Loops](README.md) |
| Wave / slice | W19 / S19.2 |
| Stories | US-20.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-09, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

The folder connector reads `.txt`, `.md` and `.pdf`; most company shares are mostly Word.

## Acceptance criteria

- [x] The folder connector reads `.docx`, `.pptx`, `.odt` and `.html`, text only, with the standard library
- [x] An archive with too many entries or too much uncompressed text is skipped, never half-read
- [x] Scripts, styles, macros and embedded objects are never read

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Extractors with fixtures
- [x] Fuzz target
- [x] Example share
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built 2026-11-23. `internal/adapters/connectors/folder/office.go`: `OfficeText` reads a .docx (`word/document.xml`), a .pptx (its slides in order, not the notes), an .odt (`content.xml`'s paragraphs and headings) and a page (.html, .htm) as text with the standard library only. No other part of an archive is opened (macros, embedded objects, media); embedded objects, deleted text, ODT scripts, forms and annotations, and a page's scripts, styles, frames, objects, SVG, comments and title are skipped. Over 4096 entries, text parts past 16 MiB (declared or read), an archive that doesn't open or a part that doesn't parse, and a page that isn't UTF-8 are skipped whole, so the file is indexed by name only. Fuzz target `FuzzOfficeText` (in `make fuzz`); the example share gains `operations/fuel-cards.docx` and `operations/night-shift.html`. Laws: `TestLAW09_OfficeFilesAreData` (LAW-09's ingest law through a real share of office files, each carrying a canary in its macros, objects, notes, scripts and styles; a self-test catches a reader that indexes them) and `TestLAW13_OfficeIndexIsDerived`. Manual: sb-connector(3).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
