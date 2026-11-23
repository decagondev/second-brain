---
title: "SB-2206: Spreadsheets and CSV on file shares"
summary: "Spreadsheets on file shares aren't read."
---

# SB-2206: Spreadsheets and CSV on file shares

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.4 |
| Stories | US-22.4 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-09, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

Spreadsheets on file shares aren't read.

## Acceptance criteria

- [x] The folder connector reads `.xlsx`, `.ods` and `.csv` as "Sheet: name" then one line a row, cells tab-separated, up to 5,000 rows and 16 MiB
- [x] Formulas are read as their cached values, never evaluated; macros are never opened
- [x] Fixtures per format and a fuzz target; a sheet past the bound is skipped whole

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Readers
- [x] Fuzz target
- [x] Law fixture formats
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: built in S21.4. `folder.SheetText` reads `.xlsx` (workbook, relationships, shared strings and worksheets only), `.ods` (`content.xml`'s tables, repeats expanded) and `.csv` (UTF-8, its delimiter guessed from the first line) as "Sheet: NAME" and one tab-separated line a row. Cells are the saved values; formulas, macros, comments, charts, drawings, external links, scripts and annotations are never read. A sheet past 5,000 rows is skipped whole, a file past 16 MiB is indexed by name only. Fixtures per format (`sheets_test.go`), `FuzzSheetText` in `make fuzz`, and the office law fixture writes all three formats with a canary in their hidden parts (`TestLAW09_OfficeFilesAreData`, `TestLAW13_OfficeIndexIsDerived`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
