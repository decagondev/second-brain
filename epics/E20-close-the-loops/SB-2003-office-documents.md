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
| Status | Open |

## Why

The folder connector reads `.txt`, `.md` and `.pdf`; most company shares are mostly Word.

## Acceptance criteria

- [ ] The folder connector reads `.docx`, `.pptx`, `.odt` and `.html`, text only, with the standard library
- [ ] An archive with too many entries or too much uncompressed text is skipped, never half-read
- [ ] Scripts, styles, macros and embedded objects are never read

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Extractors with fixtures
- [ ] Fuzz target
- [ ] Example share
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
