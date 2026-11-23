---
title: "SB-2104: Watch a saved question"
summary: "A saved question doesn't say when its sources change."
---

# SB-2104: Watch a saved question

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.2 |
| Stories | US-21.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-2101 |
| Status | Open |

## Why

A saved question doesn't say when its sources change.

## Acceptance criteria

- [ ] A saved question gains Watch; weekly the cell re-asks it as the person and compares the cited documents and versions with last week's
- [ ] When they changed, the person gets "A source of your saved question changed" and a link; a changed cite they can no longer open counts but isn't named
- [ ] No answer is stored, only the cite set's hash

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Watch flag on saved questions
- [ ] Weekly re-ask
- [ ] UI toggle
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
