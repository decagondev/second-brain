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
| Status | In review |

## Why

Model spend is predictable per tenant.

## Acceptance criteria

- [ ] Per-tenant monthly budget and rate limit from the manifest (`budget_usd_month` from the manifest, 30 calls per user per minute fixed; wiring the guard into a running cell is W7)
- [x] Budget breach degrades to refusal with `EBUDGET`, never to unmetered use

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Budget field
- [ ] Metering
- [ ] Limiter
- [ ] Error code
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.3a): `internal/core/budget.Guard` checks, before every model call, a per-user token bucket (30 a minute) and the month's spend plus the call's worst case against `budget_usd_month`, and meters the provider's reported usage after. An unreadable meter refuses the call (`EUNAVAIL`), and refusals are counted by reason. The gateway takes the guard and refuses without calling, so a test shows the provider isn't reached. `ModelRequest.User` carries the asker for the rate limit and is never sent to a provider. The ask and draft paths pass it, and `EBUDGET` maps to HTTP 429.

**Left:** a persistent meter and the guard in a running cell (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
