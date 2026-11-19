---
title: "SB-1201: Stewards and delegated space changes"
summary: "Department leads know their space best; IT shouldn't be the bottleneck, and leads mustn't be able to widen access."
---

# SB-1201: Stewards and delegated space changes

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.1 Stewards |
| Stories | US-12.1 |
| Manual | `sb-spaces(7)` (new), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | [LAW-15](../../manual/sb-laws.7.md#law-15), [LAW-16](../../manual/sb-laws.7.md#law-16), LAW-19 (new, W8) |
| Depends on | [SB-1001](../E10-department-spaces/SB-1001-spaces-in-the-manifest-and-sb-tenant-check.md), [SB-602](../E07-rollout-ops/SB-602-admin-console-where-every-action-is-a.md) |
| Status | Open |

## Why

Department leads know their space best; IT shouldn't be the bottleneck, and leads mustn't be able to widen access.

## Acceptance criteria

- [ ] A steward proposes a change to their own space in the console as a manifest diff
- [ ] The tenant operator approves the exact diff (hash-bound, LAW-15); the steward can't approve their own
- [ ] A diff that would permit past the source or remove a barrier is refused at check time

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Steward role in Cedar
- [ ] Console flow
- [ ] Check-time refusals
- [ ] Write or extend the law tests: LAW-15, LAW-16, LAW-19
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
