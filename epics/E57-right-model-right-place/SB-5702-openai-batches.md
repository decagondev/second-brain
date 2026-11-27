---
title: "SB-5702: OpenAI batches"
summary: "OpenAI-compatible providers never batch, so work that could wait pays full price."
---

# SB-5702: OpenAI batches

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.1 |
| Stories | US-57.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-05, LAW-07 |
| Depends on | SB-5604, SB-305 |
| Status | Open |

## Why

OpenAI-compatible providers never batch, so work that could wait pays full price.

## Acceptance criteria

- [ ] The `openaicompat` adapter implements `ports.Batcher` against the file-based Batch API: JSONL upload with purpose `batch`, a batch with the 24-hour window, output and error files collected
- [ ] The input and output files are deleted once collected
- [ ] sb-cell(5) `model.batch: true` says a provider supports it; without it every call runs as before

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Batcher in the OpenAI adapter
- [ ] Recorded OpenAI batch fake
- [ ] `model.batch` setting
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
