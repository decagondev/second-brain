---
title: "SB-5501: Context panel"
summary: "A person can't see what an answer was built from without reading a trace."
---

# SB-5501: Context panel

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.1 |
| Stories | US-55.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-4606 |
| Status | Done |

## Why

A person can't see what an answer was built from without reading a trace.

## Acceptance criteria

- [x] **Context** beside each answer lists passages and documents, memory lines, the skill, the collection or documents asked about, earlier questions kept and dropped, departments consulted, passages skipped and the model tier
- [x] The panel matches the trace
- [x] Titles the person can't open now are counted, not named

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Context from the trace, checked as the person
- [x] Web: the panel
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. The trace now keeps what steered an ask (`Trace.Collection`, `Documents`, `Kept`, `Dropped`) and the answer the memory lines it was given (`Response.Memory`, ids only). `Agent.Context` at `GET /v1/traces/{id}/context` builds the panel from the trace as How this was made does (SB-4606): sources, departments, skill, tier, kept and dropped, earlier questions read and left out, skipped and held, every title checked as the viewer now and the rest counted; the memory lines only for the asker. Web: **Context** under each answer (`Steer.tsx`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
