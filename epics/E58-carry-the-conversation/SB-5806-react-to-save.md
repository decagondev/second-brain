---
title: "SB-5806: React to save"
summary: "A useful chat thread is saved for stewards only by a command few people know."
---

# SB-5806: React to save

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.2 |
| Stories | US-58.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-3703, SB-2005 |
| Status | Done |

## Why

A useful chat thread is saved for stewards only by a command few people know.

## Acceptance criteria

- [x] sb-cell(5) `chat.save_reaction` (default `brain`); adding it to a thread sends the person one DM offering **Save for the stewards** or **Ask about this thread**
- [x] Nothing is read beyond the reaction until they choose; the thread's text is data
- [x] A thread in a channel the app isn't in is never read

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Reaction events in `slackapp` and `mmapp`
- [x] DM with two actions
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built for Slack. sb-cell(5) `chat.save_reaction` (default `brain`): a `reaction_added` in a channel the bot is in sends one direct message with **Save for the stewards** and **Ask about this thread** (`DM.DirectBlocks`); nothing else is read until a press, which saves the thread as *Save to the brain* does or asks about it with the thread as a file read for that ask alone (data, LAW-09), as the person in the channel's space. Mattermost sends no reaction events to an outgoing webhook, so there `/brain save LINK` stays the way to save a thread; recorded in sb-ask(3). Tested in TestReactToSave.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
