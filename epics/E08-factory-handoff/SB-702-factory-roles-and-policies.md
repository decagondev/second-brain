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
| Status | Done |

## Why

Factory operators and tenant owners are roles, never superusers.

## Acceptance criteria

- [x] Roles `factory-operator` and `tenant-owner` defined in Cedar
- [x] No role can read tenant content; factory actions are control-plane calls, audited

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Role policies
- [x] Owner designation via the tenant's IdP group
- [x] Tests
- [x] Write or extend the law tests: LAW-02, LAW-17
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.1b): sb-policy(5) gained "Factory roles, exactly". `factory-operator` is the `sb-factory-operators` group of the factory's own tenant, acting on `CellCustody` records (a new entity whose `tenant` is the factory's tenant, and whose `cell` is the tenant the record is about). `tenant-owner` is the tenant IdP's `sb-owners` group, acting on its own `Cell`. Both are base Cedar permits with the tenant condition. A forbid stops anyone approving a plan they made, and two forbids keep both roles from reading any other tenant's content. The policy kernel no longer fills an empty `tenant` for a `CellCustody`: it returns `Unknown`. Without that, a tenant with its own group named `sb-factory-operators` could act on a custody record it built itself, and a test now shows that case refused. `TestFactoryRoles` covers 17 cases through the real kernel, and every decision is recorded before it's returned. The LAW-02 policy law test now draws custody verbs, custody records and the three role groups, and finds no cross-tenant allow. LAW-17's own tests are SB-701's, red until SB-705.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
