---
title: "SB-5206: Branch a conversation"
summary: "Trying a different angle on an earlier question loses the conversation that followed it."
---

# SB-5206: Branch a conversation

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.2 |
| Stories | US-52.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | SB-2901, SB-3305, SB-2405 |
| Status | Done |

## Why

Trying a different angle on an earlier question loses the conversation that followed it.

## Acceptance criteria

- [x] **Ask from here** on an earlier question starts a branch with the context up to that point
- [x] Each branch keeps its own last 8 questions and its own download
- [x] Branches live only in the browser; nothing is stored on the server

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Branch in the UI
- [x] Context per branch
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `Agent.Branch` at `POST /v1/conversations/ID/branch` `{at}`: a new conversation of the caller's own with the first N questions copied (never an answer), tagged with the space, the first left as it was. Web: **Ask from here** on an earlier answer branches, and the conversation it left is kept in the browser under **Other branches** to go back to; each branch downloads on its own. The branch is a conversation on the server, like any other, rather than in the browser alone: the server keeps the earlier questions a turn sends, so a branch needs its own. Tested in TestBranchAConversation and web `branch.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
