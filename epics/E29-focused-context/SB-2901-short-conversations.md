---
title: "SB-2901: Conversations stay short"
summary: "A long conversation sends every earlier question to the model, however far the topic moved."
---

# SB-2901: Conversations stay short

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.1 |
| Stories | US-29.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Open |

## Why

A long conversation sends every earlier question to the model, however far the topic moved.

## Acceptance criteria

- [ ] A later turn sends at most the last 8 earlier questions; the response says `dropped` when some were left out
- [ ] The UI offers Start fresh then; the stored conversation isn't changed
- [ ] The ninth earlier question isn't in the prompt; Start fresh starts a new conversation

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ask
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
