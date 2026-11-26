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
| Status | Open |

## Why

Briefs, routines, checks and evals that could wait for tomorrow are paid at full price now.

## Acceptance criteria

- [ ] Scheduled briefs, routines, checks and evals go to the provider's batch API where the adapter has one (Anthropic, OpenAI), submitted by the notify job and collected on the next tick, with results by the next working morning
- [ ] Anything a person is waiting for never batches; a provider without a batch API runs them as now
- [ ] A failed batch reruns once, then runs normally

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Batch submit and collect in the adapters
- [ ] Notify job ticks
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
