---
title: "SB-3703: Save a chat thread"
summary: "A thread that settles something in chat is lost to the brain."
---

# SB-3703: Save a chat thread

| Field | Value |
|---|---|
| Epic | [E37 Capture Conversations](README.md) |
| Wave / slice | W36 / S36.1 |
| Stories | US-37.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-11, LAW-12, LAW-16 |
| Depends on | SB-1705 |
| Status | Open |

## Why

A thread that settles something in chat is lost to the brain.

## Acceptance criteria

- [ ] A Slack or Mattermost message action, Save to the brain, sends the thread to the space's stewards as a draft note
- [ ] Publish as a note runs the note flow; Discard drops it
- [ ] Only people the cell knows can send one; audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Chat apps
- [ ] Queue and steward screen
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
