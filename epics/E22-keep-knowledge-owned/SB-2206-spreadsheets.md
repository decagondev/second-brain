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
| Status | Open |

## Why

Spreadsheets on file shares aren't read.

## Acceptance criteria

- [ ] The folder connector reads `.xlsx`, `.ods` and `.csv` as "Sheet: name" then one line a row, cells tab-separated, up to 5,000 rows and 16 MiB
- [ ] Formulas are read as their cached values, never evaluated; macros are never opened
- [ ] Fixtures per format and a fuzz target; a sheet past the bound is skipped whole

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Readers
- [ ] Fuzz target
- [ ] Law fixture formats
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
