---
title: "SB-5605: Cache the stable part"
summary: "The stable part of every prompt is sent and billed in full on every call."
---

# SB-5605: Cache the stable part

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.2 |
| Stories | US-56.2 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-02, LAW-07 |
| Depends on | SB-305 |
| Status | Done |

## Why

The stable part of every prompt is sent and billed in full on every call.

## Acceptance criteria

- [x] Prompts put the stable part first (system prompt, agent and skill instructions, glossary); adapters mark it cacheable where the provider supports it
- [x] Traces record cached and uncached tokens, and costs use the provider's cached price
- [x] No passage, memory, file or question is ever in the cached prefix

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Prompt order and cache marks
- [x] Cached tokens in traces and costs
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. The answer's system prompt is its stable part and carries `ModelRequest.CacheSystem`; nothing the asker sent is in it (TestStablePrefixAndBatching). Anthropic's adapter sends it as a block with `cache_control` and reads `cache_read_input_tokens`, the OpenAI-compatible adapter reads `prompt_tokens_details.cached_tokens`, into `Usage.Cached`; `budget.Price.Cost` charges them at sb-cell(5) `model.price.cached_in_per_m` through `Guard.RecordCost` and the tally.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
