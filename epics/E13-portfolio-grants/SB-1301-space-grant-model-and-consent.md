---
title: "SB-1301: Space grant model and consent"
summary: "ADR-0014 allows parent access only as a separate tenant with consent; spaces let that consent be narrow."
---

# SB-1301: Space grant model and consent

| Field | Value |
|---|---|
| Epic | [E13 Portfolio Department Grants](README.md) |
| Wave / slice | W13 / S13.1 Grants |
| Stories | US-13.1 |
| Manual | `sb-spaces(7)` (new), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), LAW-23 (new, W8) |
| Depends on | [SB-1201](../E12-space-governance/SB-1201-stewards-and-delegated-space-changes.md) |
| Status | Parked (Q-01 stays no, 2026-11-20) |

## Why

ADR-0014 allows parent access only as a separate tenant with consent; spaces let that consent be narrow.

## Acceptance criteria

- [ ] Starts only once Q-01 is answered yes and ADR-0014 is Accepted
- [ ] A grant names one company space, one parent space, a report type and an expiry; it needs the company owner and company legal (dual approval)
- [ ] Recorded in both cells' audit chains

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] ADR-0014 acceptance
- [ ] Grant record
- [ ] Consent flow
- [ ] Audit rows both sides
- [ ] Write or extend the law tests: LAW-02, LAW-23
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
