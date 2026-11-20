---
title: "SB-1302: Aggregate-only grant delivery"
summary: "The parent needs numbers and statements, not documents."
---

# SB-1302: Aggregate-only grant delivery

| Field | Value |
|---|---|
| Epic | [E13 Portfolio Department Grants](README.md) |
| Wave / slice | W13 / S13.1 Grants |
| Stories | US-13.1 |
| Manual | `sb-spaces(7)` (new) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), LAW-23 (new, W8) |
| Depends on | [SB-1301](SB-1301-space-grant-model-and-consent.md) |
| Status | Parked (Q-01 stays no, 2026-11-20) |

## Why

The parent needs numbers and statements, not documents.

## Acceptance criteria

- [ ] The company cell computes the approved report and sends only it; no chunk, citation or document id crosses
- [ ] Each delivery is hashed and audited on both sides
- [ ] LAW-23 green; LAW-02 still green

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Report runner
- [ ] Delivery channel
- [ ] LAW-23 law test green
- [ ] Write or extend the law tests: LAW-02, LAW-23
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
