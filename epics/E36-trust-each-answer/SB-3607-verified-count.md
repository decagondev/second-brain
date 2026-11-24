---
title: "SB-3607: How sure the answer is"
summary: "An answer doesn't say how many of its sources someone has checked."
---

# SB-3607: How sure the answer is

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.2 |
| Stories | US-36.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14 |
| Depends on | SB-2702 |
| Status | Done |

## Why

An answer doesn't say how many of its sources someone has checked.

## Acceptance criteria

- [x] The response's `verified` counts the cited documents and those with a live mark at the cited version
- [x] The UI shows N of M sources verified, and chat says it
- [x] No cites, no line; a stale mark doesn't count

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent
- [x] UI and chat
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

The response's verified count, N of M sources verified in the UI and the chat apps

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
