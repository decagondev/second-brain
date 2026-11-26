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
| Status | Open |

## Why

A person can't see what an answer was built from without reading a trace.

## Acceptance criteria

- [ ] **Context** beside each answer lists passages and documents, memory lines, the skill, the collection or documents asked about, earlier questions kept and dropped, departments consulted, passages skipped and the model tier
- [ ] The panel matches the trace
- [ ] Titles the person can't open now are counted, not named

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Context from the trace, checked as the person
- [ ] Web: the panel
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
