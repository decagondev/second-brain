---
title: "SB-5703: People first when the provider is busy"
summary: "When a provider rate-limits, scheduled work and people's questions compete equally, and one busy space can starve the rest."
---

# SB-5703: People first when the provider is busy

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.1 |
| Stories | US-57.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-03, LAW-14 |
| Depends on | SB-5601, SB-5602 |
| Status | Done |

## Why

When a provider rate-limits, scheduled work and people's questions compete equally, and one busy space can starve the rest.

## Acceptance criteria

- [x] On a rate limit the gateway queues calls per space, serves a person's ask before any scheduled step, and serves spaces in turn
- [x] Scheduled work that can't run waits for the next tick; an ask that would wait more than 20 seconds gets passages with "the model is busy"
- [x] The health report counts rate limits and waits per space

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Per-space queue in the gateway
- [x] Passages after 20 seconds
- [x] Health counts
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `gateway.Busy` is one queue per provider and base (shared by its tiers, kept on the cell across applies). Outside a rate limit calls run side by side; after a 429 calls take turns until one goes through with nobody waiting: asks before scheduled calls, then the space served least in the storm, then the earliest. A deferred scheduled step goes to the next tick (`ErrBatchPending`), other scheduled calls wait behind asks and retry as before, and an ask past 20 seconds is `gateway.ErrBusy`, which the ask answers with passages and `notice: "model_busy"` (web text in English, German and French). The health report's `model.busy` counts rate limits and waits per space, with `sb_model_rate_limited_total`. Tested in TestPeopleFirstWhenBusy, TestAskPastTheWaitIsBusy and TestBusyModelGivesPassages.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
