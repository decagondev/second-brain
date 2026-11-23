---
title: "SB-2207: National ID numbers as personal data"
summary: "Personal data flags know IBANs and cards, not national ID numbers."
---

# SB-2207: National ID numbers as personal data

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.4 |
| Stories | US-22.4 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Done |

## Why

Personal data flags know IBANs and cards, not national ID numbers.

## Acceptance criteria

- [x] The personal-data scanner gains UK NINOs, US SSNs, Dutch BSNs (11-check) and German tax ids (ISO 7064)
- [x] They are flagged `personal_data` and masked but the last four
- [x] Valid and near-miss fixtures per format; the masking law test covers them

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Detectors
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: built in S21.4. `ingest.nationalIDs` finds UK NINOs (prefix and suffix rules), US SSNs (area, group and serial rules, separators required), Dutch BSNs (11-check; 9 digits or 4.2.3) and German tax ids (the repeated-digit rule and ISO 7064 MOD 11,10). They flag `personal_data` and mask but the last four like IBANs and cards. Valid and near-miss fixtures in `nationalid_test.go`; the PII law's numbers include them, a self-test catches a masker that knows only IBANs and cards, and `FuzzMaskPII` is in `make fuzz`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
