---
title: "SB-806: Threat model v1 for spaces and ADR-0020, ADR-0021 sign-off"
summary: "Spaces add new ways to leak inside a company: mislabelled files, over-broad scopes, a company-wide ask that pulls HR content."
---

# SB-806: Threat model v1 for spaces and ADR-0020, ADR-0021 sign-off

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.2 Laws red, model green |
| Stories | US-00.7 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-19 (new, W8), LAW-20 (new, W8), LAW-22 (new, W8) |
| Depends on | [SB-804](SB-804-law-19-to-law-23-as-failing-tests.md) |
| Status | Open |

## Why

Spaces add new ways to leak inside a company: mislabelled files, over-broad scopes, a company-wide ask that pulls HR content.

## Acceptance criteria

- [ ] `docs/07-threat-model.md` gains the inside-the-company threats, each mapped to a law and a test
- [ ] ADR-0020 and ADR-0021 signed off by two reviewers (human)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Threat walk-through
- [ ] Mapping to laws
- [ ] Sign-off rows
- [ ] Write or extend the law tests: LAW-19, LAW-20, LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
