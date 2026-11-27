---
title: "SB-5705: Keep model calls in region"
summary: "Nothing stops a model call from leaving the region a company's data must stay in."
---

# SB-5705: Keep model calls in region

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.2 |
| Stories | US-57.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-03, LAW-33 |
| Depends on | SB-5601, SB-5604, SB-5007 |
| Status | Done |

## Why

Nothing stops a model call from leaving the region a company's data must stay in.

## Acceptance criteria

- [x] sb-cell(5) `model.region` and a `region` on every provider entry (main, small, deep, fallback, a remote embedder)
- [x] With `model.region` set, no call of any tier, batch or fallback goes to a provider of another region or of none; a failing in-region provider gives passages
- [x] The health report and the inventory show each provider's region; LAW-33 *a model call stays in its region* is green

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Region check in the gateway
- [x] LAW-33 law test and selftest
- [x] Health and inventory lines
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-cell(5) `model.region`, with `model.provider_region`, `model.fallback.region`, `embed.region` and `embed.local`. Every gateway checks `gateway.InRegion` before a call or batch leaves and refuses with `ErrRegion` (unavailable, so an ask shows passages; `sb_model_refusals_total{reason="region"}`); a local model is in every region, and a provider with no region is never in one (LAW-03). A remote embedder outside the region stops the cell at start. The health report's `model.regions` and the inventory's `models` list every provider with its region and `in_region`. LAW-33 is written in sb-laws(7) and green: TestLAW33_AModelCallStaysInItsRegion with TestSelfRegionBlindGatewayIsCaught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
