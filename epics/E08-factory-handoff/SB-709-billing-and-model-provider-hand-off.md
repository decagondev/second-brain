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
| Status | Open |

## Why

The company must be able to pay for and contract its own model use.

## Acceptance criteria

- [ ] Per-tenant cost statements; option for a tenant-held model provider credential behind the `Model` port

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Cost statement export
- [ ] Tenant-held model credential via vault
- [ ] Docs
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
