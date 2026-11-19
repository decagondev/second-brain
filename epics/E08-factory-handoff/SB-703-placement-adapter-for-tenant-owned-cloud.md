---
title: "SB-703: Placement adapter for tenant-owned cloud accounts"
summary: "Stamp a cell into a company's own account with least privilege."
---

# SB-703: Placement adapter for tenant-owned cloud accounts

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.1 Custody model |
| Stories | US-08.2 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-114](../E08-factory-handoff/SB-114-factory-seams-custody-record-placement-port.md), [SB-702](../E08-factory-handoff/SB-702-factory-roles-and-policies.md) |
| Status | Open |

## Why

Stamp a cell into a company's own account with least privilege.

## Acceptance criteria

- [ ] Cell stamped into a tenant account via a scoped deploy role the tenant creates
- [ ] Same template; no tenant-specific code

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Tenant-account adapter behind `Placement`
- [ ] Bootstrap script for the tenant's deploy role
- [ ] Two-account test
- [ ] Write or extend the law tests: LAW-02
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
