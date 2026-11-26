---
title: "SB-5502: Keep or drop for this conversation"
summary: "Steering an answer means rewording the question and hoping."
---

# SB-5502: Keep or drop for this conversation

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.1 |
| Stories | US-55.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01 |
| Depends on | SB-5501, SB-3004, SB-2405 |
| Status | Open |

## Why

Steering an answer means rewording the question and hoping.

## Acceptance criteria

- [ ] From the panel, **Keep** adds a document to every later ask in the conversation and **Drop** leaves it out
- [ ] Both live with the conversation in the browser and go into its download; nothing is stored on the server
- [ ] A kept document is checked as the person on each ask and stops being used if they lose access

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Web: keep and drop
- [ ] Request carries kept and dropped documents
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
