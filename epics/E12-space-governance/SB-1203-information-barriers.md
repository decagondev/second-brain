---
title: "SB-1203: Information barriers"
summary: "Deal teams, investigations and some legal matters must not see, or be seen by, the rest of the company."
---

# SB-1203: Information barriers

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.2 Barriers and duties |
| Stories | US-12.2 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md), `sb-space(5)` (new) |
| Laws | LAW-22 (new, W8) |
| Depends on | [SB-1004](../E10-department-spaces/SB-1004-cedar-schema-v2-spaces-labels-narrowing.md) |
| Status | Open |

## Why

Deal teams, investigations and some legal matters must not see, or be seen by, the rest of the company.

## Acceptance criteria

- [ ] `barriers:` pairs of groups become Cedar forbids in both directions, for reads, drafts and actions
- [ ] A document owned by a barred group is invisible across the barrier, even company-wide and even if shared in the source
- [ ] LAW-22 green

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Policy generation
- [ ] Ownership from the source
- [ ] LAW-22 law test green
- [ ] Write or extend the law tests: LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
