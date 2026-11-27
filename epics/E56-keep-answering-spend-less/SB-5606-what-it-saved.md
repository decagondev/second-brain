---
title: "SB-5606: What it saved"
summary: "Nobody can see what reuse, batching, caching and the economy tier are worth."
---

# SB-5606: What it saved

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.2 |
| Stories | US-56.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-11 |
| Depends on | SB-3505, SB-5006, SB-5603, SB-5604, SB-5605 |
| Status | Done |

## Why

Nobody can see what reuse, batching, caching and the economy tier are worth.

## Acceptance criteria

- [x] The cost statement and the admin's week gain a "saved" line: reuse, batching, caching and the economy tier, each in dollars this month
- [x] `GET /v1/admin/costs/saved` gives the same as JSON, worked out from traces against the full price
- [x] A month with no savings says so

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Savings from traces
- [x] Statement and admin's week lines
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. The gateway's `spent` saves into the answer's tally what caching, batching and the economy tier (`Gateway.MainPrice` on the small tier) took off, and a reused answer saves its earlier cost; `Trace.Saved` keeps it. `costs.AddSaved` sums a month's at `GET /v1/admin/costs/saved` (operators), on the cost statement as `saved`, and in the admin's week; a month with none says "Nothing saved this month." Tested in TestSavedAddsUp and TestKeepAnsweringThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
