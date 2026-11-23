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
| Status | Open |

## Why

Personal data flags know IBANs and cards, not national ID numbers.

## Acceptance criteria

- [ ] The personal-data scanner gains UK NINOs, US SSNs, Dutch BSNs (11-check) and German tax ids (ISO 7064)
- [ ] They are flagged `personal_data` and masked but the last four
- [ ] Valid and near-miss fixtures per format; the masking law test covers them

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Detectors
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
