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
| Status | Done |

## Why

A long conversation sends every earlier question to the model, however far the topic moved.

## Acceptance criteria

- [x] A later turn sends at most the last 8 earlier questions; the response says `earlier_dropped` when some were left out
- [x] The UI offers Start fresh then; the stored conversation isn't changed
- [x] The ninth earlier question isn't in the prompt; Start fresh starts a new conversation

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ask
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S28.1; tested in internal/core/ask/focus_test.go, internal/core/route/report_test.go, internal/cli/scopereport_test.go, tests/cell/retire_test.go and web/src/focus.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
