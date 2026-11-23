---
title: "SB-2106: Handover by name"
summary: "The Handover tab takes a user id, which managers don't know."
---

# SB-2106: Handover by name

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.3 |
| Stories | US-21.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

The Handover tab takes a user id, which managers don't know.

## Acceptance criteria

- [ ] The Handover tab gains a person search listing, by name and email, only the people the asker may brief: themselves, their reports, members of spaces they steward
- [ ] A colleague with no relation isn't in the list, whatever they type
- [ ] Typing a user id still works

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] People endpoint
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
