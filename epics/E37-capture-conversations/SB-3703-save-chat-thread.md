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
| Status | Done |

## Why

A thread that settles something in chat is lost to the brain.

## Acceptance criteria

- [x] A Slack or Mattermost message action, Save to the brain, sends the thread to the space's stewards as a draft note
- [x] Publish as a note runs the note flow; Discard drops it
- [x] Only people the cell knows can send one; audited

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Chat apps
- [x] Queue and steward screen
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Save to the brain from a Slack shortcut or /brain save LINK; the stewards reply and publish, or discard

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
