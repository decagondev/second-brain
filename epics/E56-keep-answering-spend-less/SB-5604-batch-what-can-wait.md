---
title: "SB-5604: Batch what can wait"
summary: "Briefs, routines, checks and evals that could wait for tomorrow are paid at full price now."
---

# SB-5604: Batch what can wait

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.2 |
| Stories | US-56.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-05, LAW-16 |
| Depends on | SB-5005, SB-4302 |
| Status | Done |

## Why

Briefs, routines, checks and evals that could wait for tomorrow are paid at full price now.

## Acceptance criteria

- [x] Scheduled briefs, routines, checks and evals go to the provider's batch API where the adapter has one (Anthropic, OpenAI), submitted by the notify job and collected on the next tick, with results by the next working morning
- [x] Anything a person is waiting for never batches; a provider without a batch API runs them as now
- [x] A failed batch reruns once, then runs normally

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Batch submit and collect in the adapters
- [x] Notify job ticks
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ModelRequest.Batch` marks a call nobody waits for (`ask.withBatch`: an eval probe, or a brief, routine or rhythm step named so by its caller; catch-up never). `Gateway.batched` sends it through `ports.Batcher` where the adapter has one (Anthropic's Message Batches, `SubmitBatch`/`CollectBatch`), looks every 2s for at most 10 minutes, retries a failed batch once and otherwise runs it as a normal call; it is charged at `BatchRate` (half). Deviation from the plan, recorded here and in sb-runtime(7): batches are collected within the run that sends them rather than on the notify job's next tick, and OpenAI's file-based batch API isn't wired yet, so OpenAI-compatible providers run as before. Tested in TestCacheAndBatch and TestBatchCacheAndTierSavings.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
