---
title: "SB-5701: Collect batches later"
summary: "A batched brief holds its run open for up to 10 minutes and is lost if the cell restarts."
---

# SB-5701: Collect batches later

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.1 |
| Stories | US-57.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-05, LAW-07 |
| Depends on | SB-5604 |
| Status | Done |

## Why

A batched brief holds its run open for up to 10 minutes and is lost if the cell restarts.

## Acceptance criteria

- [x] A batchable call is submitted by one notify tick, its batch id, provider, caller and step kept in `state/batches/ID.json` (never the prompt), and collected by a later tick; the submitting run ends at once
- [x] A restart between submit and collect loses nothing
- [x] A batch not back by 06:00 on the next working morning (cell time) is cancelled and its step runs as a normal call, rebuilt from its source; a failed batch reruns once, then normally

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Batch state in `state/batches`
- [x] Collect on the notify tick
- [x] Morning deadline and cancel
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ports.WithDeferredBatch` names a scheduled step (briefs, updates and the leader's week in `notify.Job.briefs`, routines in `Legwork.runRoutine`, rhythm steps in `Rhythms.offer`); for those, `Gateway.deferred` submits the batchable call, keeps a `ports.PendingBatch` (key, batch id, provider, caller, step, submitted, deadline, tries; never the prompt) in `state/batches/KEY.json` (`wire.batchStore`) and answers `ports.ErrBatchPending`, which ask turns into `ask.ErrPending` and the steps into `notify.BriefPending` or no mark, so a later tick runs the step again, rebuilt from its source, and collects the result. `gateway.BatchDeadline` is 06:00 on the next working morning in the digest zone; past it the batch is cancelled (`ports.BatchCanceller`, Anthropic's `/cancel`) and the call runs normally, a failed batch is submitted once more, and `Job.SweepBatches` cancels batches no step came back for. Tested in TestBatchCollectedOnALaterTick, TestSweepBatches, TestBatchDeadline, TestPendingBriefRunsAgainNextTick, TestBatchPendingIsNotModelDown and the law test TestLAW07_KeptBatchesHoldNoPrompt with TestSelfBatchesAreCaught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
