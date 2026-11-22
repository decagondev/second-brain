---
title: "SB-1904: Personal data flags at ingest"
summary: "Ingest flags secrets but not personal data, so a payroll sheet in an all-staff space looks like any other document."
---

# SB-1904: Personal data flags at ingest

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.2 |
| Stories | US-19.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-13 |
| Depends on | — |
| Status | Open |

## Why

Ingest flags secrets but not personal data, so a payroll sheet in an all-staff space looks like any other document.

## Acceptance criteria

- [ ] Ingest flags chunks holding an IBAN (checksum valid) or a payment card number (Luhn valid)
- [ ] A flagged chunk's search snippet and answer quotes mask the number, keeping the last four characters
- [ ] Stewards see how many documents in their space carry personal data

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Detectors with a fuzz target
- [ ] Masking in snippets
- [ ] Space count
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
