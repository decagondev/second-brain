---
title: "SB-609: Cost controls"
summary: "Model spend is predictable per tenant."
---

# SB-609: Cost controls

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.3 Operate |
| Stories | US-07.4 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | — |
| Depends on | [SB-607](../E07-rollout-ops/SB-607-per-tenant-observability.md) |
| Status | Open |

## Why

Model spend is predictable per tenant.

## Acceptance criteria

- [ ] Per-tenant monthly budget and rate limit from the manifest
- [ ] Budget breach degrades to refusal with `EBUDGET`, never to unmetered use

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Budget field
- [ ] Metering
- [ ] Limiter
- [ ] Error code
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
