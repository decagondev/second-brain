---
title: "SB-5504: Memory used, shown"
summary: "A person can't tell when a remembered line shaped an answer."
---

# SB-5504: Memory used, shown

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.1 |
| Stories | US-55.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-25 |
| Depends on | SB-4303 |
| Status | Done |

## Why

A person can't tell when a remembered line shaped an answer.

## Acceptance criteria

- [x] An answer that used a memory line shows it under the answer with **Forget this**
- [x] The line shown is the one in the trace; an answer that used none says nothing
- [x] Forgetting it removes it from the next answer

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Memory lines on the response
- [x] Web: **Forget this**
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `Agent.memoryUsed` returns the lines an ask is given with their ids; the answer carries the ids, and the web shows each line under the answer with **Forget this** (`MemoryUsed`), which is the existing forget, so the next answer isn't given it.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
