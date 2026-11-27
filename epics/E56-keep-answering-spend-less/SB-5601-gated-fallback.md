---
title: "SB-5601: Fall back to a gated model"
summary: "One model provider failing stops every answer."
---

# SB-5601: Fall back to a gated model

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.1 |
| Stories | US-56.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-03, LAW-06 |
| Depends on | SB-305, SB-4405 |
| Status | Done |

## Why

One model provider failing stops every answer.

## Acceptance criteria

- [x] sb-cell(5) `model.fallback` names a second provider; on a server error, rate limit or timeout the ask goes there only if it passed the release's eval gate, and the answer says so
- [x] After 3 failures in 5 minutes the cell uses the fallback for 10 minutes before trying the first again, shown in the health report
- [x] An ungated fallback is never used

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Gateway fallback with a circuit
- [x] Gate check from `state/evals`
- [x] Health line
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-cell(5) `model.fallback` builds a second gateway; `gateway.Fallback` wraps the main tier: on an unavailable, rate-limited or timed-out call it goes to the fallback only while `Cell.fallbackGated` (the evals job writes `state/evals/fallback-gated` naming the release when every passing case also passes on the fallback alone, `gateway.WithFallbackOnly`), and after 3 failures in 5 minutes it uses the fallback for 10. A secret or budget refusal never falls back. `Response.Fallback` says so, and the health report's `model` shows the state. LAW-06 property `TestLAW06_OnlyAGatedFallbackAnswers` with `TestSelfUngatedFallbackIsCaught`; tested in TestFallbackOnlyWhenGated and TestKeepAnsweringThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
