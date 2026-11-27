---
title: "SB-5805: Where you left off"
summary: "Unfinished work is found only by opening the inbox."
---

# SB-5805: Where you left off

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.2 |
| Stories | US-58.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-4301, SB-4504, SB-4903 |
| Status | Done |

## Why

Unfinished work is found only by opening the inbox.

## Acceptance criteria

- [x] A new web conversation and the first DM of the day start with up to 3 open things from the last 14 days: a question waiting on a steward, a research plan to approve, a confirmed action not done, an overdue wait
- [x] Each opens where it is; everything comes from existing stores; **Not now** hides it for the day
- [x] An item the person can no longer open isn't shown

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Open things from existing stores
- [x] Web and chat-app lines
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Legwork.LeftOff` at `GET /v1/me/left-off`: up to 3 of the caller's own open things from 14 days (a question waiting on a steward, a research plan or checkpoint, a confirmed action not done whose transcript they can open, an overdue wait they set in a space they're in), from the existing stores, overdue first. The web's `LeftOff` above the ask box of a new conversation, with **Not now** for the day in the browser; Slack direct messages to the bot are now answered there, and the first of the day starts with the same lines. Tested in TestCarryTheConversationThroughARunningCell, TestDirectMessageStartsWhereYouLeftOff and the web's carry test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
