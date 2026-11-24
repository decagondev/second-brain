---
title: "SB-3304: Sources disagree"
summary: "When two sources disagree, the answer picks one quietly."
---

# SB-3304: Sources disagree

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.1 |
| Stories | US-33.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Open |

## Why

When two sources disagree, the answer picks one quietly.

## Acceptance criteria

- [ ] A claim may carry `disagree: true`, kept only when it cites at least two different documents the asker can open
- [ ] The UI shows Sources disagree before the claim and chat says it in words
- [ ] The citation suite gains a disagreeing and an agreeing case, and they pass

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Answer schema
- [ ] UI and chat
- [ ] Evals
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
