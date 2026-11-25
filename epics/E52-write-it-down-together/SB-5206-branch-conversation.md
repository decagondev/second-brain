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
| Status | Open |

## Why

Trying a different angle on an earlier question loses the conversation that followed it.

## Acceptance criteria

- [ ] **Ask from here** on an earlier question starts a branch with the context up to that point
- [ ] Each branch keeps its own last 8 questions and its own download
- [ ] Branches live only in the browser; nothing is stored on the server

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Branch in the UI
- [ ] Context per branch
- [ ] Tests
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
