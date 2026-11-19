---
title: "SB-1205: SCIM attributes and five-minute moves"
summary: "When someone changes team, their access should change with them, quickly."
---

# SB-1205: SCIM attributes and five-minute moves

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.3 Reorgs |
| Stories | US-12.3 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | [LAW-12](../../manual/sb-laws.7.md#law-12), LAW-21 (new, W8) |
| Depends on | [SB-107](../E01-tenancy-identity/SB-107-scim-provisioning-and-fast-deprovisioning.md) |
| Status | Open |

## Why

When someone changes team, their access should change with them, quickly.

## Acceptance criteria

- [ ] SCIM enterprise attributes (`department`, `division`, `costCenter`, `manager`) and title level are stored and passed to Cedar
- [ ] Leaving a space's group revokes that space within 5 minutes on a fake clock; joining grants only after the cell confirms the membership
- [ ] LAW-21 green

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Attribute mapping
- [ ] Move handling
- [ ] LAW-21 law test green
- [ ] Write or extend the law tests: LAW-12, LAW-21
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
