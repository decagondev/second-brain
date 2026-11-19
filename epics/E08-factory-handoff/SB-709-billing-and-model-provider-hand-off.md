---
title: "SB-709: Billing and model-provider hand-off"
summary: "The company must be able to pay for and contract its own model use."
---

# SB-709: Billing and model-provider hand-off

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.3 Exit paths |
| Stories | US-08.5 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | — |
| Depends on | [SB-609](../E07-rollout-ops/SB-609-cost-controls.md) |
| Status | In review |

## Why

The company must be able to pay for and contract its own model use.

## Acceptance criteria

- [ ] Per-tenant cost statements; option for a tenant-held model provider credential behind the `Model` port (both built; a persistent per-model usage meter in a running cell is W7's runtime)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Cost statement export
- [x] Tenant-held model credential via vault
- [x] Docs
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.3b): sb-tenant(5) gained "Model billing and cost statements". Whoever's key is at `vault://tenants/TENANT/model` holds the provider contract. A company switches to its own account by putting its key there, with no deploy and no manifest change, while the release still fixes provider and model. sb-admin(3) gained `GET /v1/admin/costs`, and sb-factory(7) notes that the model contract moves at the tenant's pace. `internal/core/costs` builds the month's statement from metered usage and the release's prices: counts and prices only, `partial` for a running month, and an error rather than a cost when a price or the meter is missing. The admin endpoint serves it as JSON or CSV to operators and approvers. A `readers` guard is now shared with `usage`. `TestTenantHeldKeyIsUsedFromTheNextCall` (openaicompat) shows a key rotated in the vault is used on the very next call and never crosses to another cell.

**Left:** a persistent per-provider, per-model usage meter in a running cell (W7 runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
