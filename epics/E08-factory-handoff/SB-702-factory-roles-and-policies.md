---
title: "SB-702: Factory roles and policies"
summary: "Factory operators and tenant owners are roles, never superusers."
---

# SB-702: Factory roles and policies

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.1 Custody model |
| Stories | US-08.1 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-603](../E07-rollout-ops/SB-603-roles-in-the-console.md), [SB-114](../E08-factory-handoff/SB-114-factory-seams-custody-record-placement-port.md) |
| Status | Open |

## Why

Factory operators and tenant owners are roles, never superusers.

## Acceptance criteria

- [ ] Roles `factory-operator` and `tenant-owner` defined in Cedar
- [ ] No role can read tenant content; factory actions are control-plane calls, audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Role policies
- [ ] Owner designation via the tenant's IdP group
- [ ] Tests
- [ ] Write or extend the law tests: LAW-02, LAW-17
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
