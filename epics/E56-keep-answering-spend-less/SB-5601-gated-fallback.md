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
| Status | Open |

## Why

One model provider failing stops every answer.

## Acceptance criteria

- [ ] sb-cell(5) `model.fallback` names a second provider; on a server error, rate limit or timeout the ask goes there only if it passed the release's eval gate, and the answer says so
- [ ] After 3 failures in 5 minutes the cell uses the fallback for 10 minutes before trying the first again, shown in the health report
- [ ] An ungated fallback is never used

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Gateway fallback with a circuit
- [ ] Gate check from `state/evals`
- [ ] Health line
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
