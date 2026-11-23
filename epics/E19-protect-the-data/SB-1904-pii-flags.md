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
| Status | Done |

## Why

Ingest flags secrets but not personal data, so a payroll sheet in an all-staff space looks like any other document.

## Acceptance criteria

- [x] Ingest flags chunks holding an IBAN (checksum valid) or a payment card number (Luhn valid)
- [x] A flagged chunk's search snippet and answer quotes mask the number, keeping the last four characters
- [x] Stewards see how many documents in their space carry personal data

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Detectors with a fuzz target
- [x] Masking in snippets
- [x] Space count
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.2): ingest flags `personal_data` on every chunk that holds, or touches, an IBAN with valid check digits (mod-97) or a card number that passes Luhn and has a card network prefix (`internal/core/ingest/pii.go`, with `FuzzMaskPII`). A flagged chunk is indexed and citable, but its search snippet and the text the model is given are masked but the last four characters (`ingest.Shown`), and every answer text is masked too, so an answer echoing the number keeps four characters. `GET /v1/spaces/NAME/review` carries `personal_data`, the count of documents in the space with the flag (`DocQuery.Flag` in both indexes), and the stewards' review list in the UI shows it. `TestLAW07_PersonalDataIsMasked` checks flags, masking, the model context and an echoing answer, and that a rebuild gives the same flags (LAW-13); a self-test catches three broken maskers.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
