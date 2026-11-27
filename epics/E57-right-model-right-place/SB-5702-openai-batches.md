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
| Status | Done |

## Why

OpenAI-compatible providers never batch, so work that could wait pays full price.

## Acceptance criteria

- [x] The `openaicompat` adapter implements `ports.Batcher` against the file-based Batch API: JSONL upload with purpose `batch`, a batch with the 24-hour window, output and error files collected
- [x] The input and output files are deleted once collected
- [x] sb-cell(5) `model.batch: true` says a provider supports it; without it every call runs as before

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Batcher in the OpenAI adapter
- [x] Recorded OpenAI batch fake
- [x] `model.batch` setting
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `openaicompat.Batching` implements `ports.Batcher` and `ports.BatchCanceller` over OpenAI's Files and Batches endpoints: one JSONL line uploaded with purpose `batch`, a batch with the 24-hour window, the output (or error) file read, and the input, output and error files deleted once collected or cancelled. sb-cell(5) `model.batch: true` builds it for an `openaicompat` provider; otherwise the plain `Client`, which has no batch API, runs every call as before; the setting is refused for any other provider and a local model. Tested against a recorded fake in TestOpenAIBatch; the batch rate is the gateway's, as for Anthropic.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
