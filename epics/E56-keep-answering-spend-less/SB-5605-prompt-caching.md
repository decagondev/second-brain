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
| Status | Open |

## Why

The stable part of every prompt is sent and billed in full on every call.

## Acceptance criteria

- [ ] Prompts put the stable part first (system prompt, agent and skill instructions, glossary); adapters mark it cacheable where the provider supports it
- [ ] Traces record cached and uncached tokens, and costs use the provider's cached price
- [ ] No passage, memory, file or question is ever in the cached prefix

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Prompt order and cache marks
- [ ] Cached tokens in traces and costs
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
