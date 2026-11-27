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
| Status | Open |

## Why

Nothing stops a model call from leaving the region a company's data must stay in.

## Acceptance criteria

- [ ] sb-cell(5) `model.region` and a `region` on every provider entry (main, small, deep, fallback, a remote embedder)
- [ ] With `model.region` set, no call of any tier, batch or fallback goes to a provider of another region or of none; a failing in-region provider gives passages
- [ ] The health report and the inventory show each provider's region; LAW-33 *a model call stays in its region* is green

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Region check in the gateway
- [ ] LAW-33 law test and selftest
- [ ] Health and inventory lines
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
