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
| Status | Open |

## Why

A batched brief holds its run open for up to 10 minutes and is lost if the cell restarts.

## Acceptance criteria

- [ ] A batchable call is submitted by one notify tick, its batch id, provider, caller and step kept in `state/batches/ID.json` (never the prompt), and collected by a later tick; the submitting run ends at once
- [ ] A restart between submit and collect loses nothing
- [ ] A batch not back by 06:00 on the next working morning (cell time) is cancelled and its step runs as a normal call, rebuilt from its source; a failed batch reruns once, then normally

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Batch state in `state/batches`
- [ ] Collect on the notify tick
- [ ] Morning deadline and cancel
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
