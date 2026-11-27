---
title: "SB-5802: Continue on the web"
summary: "A conversation started in a chat app can't be continued on the web."
---

# SB-5802: Continue on the web

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.1 |
| Stories | US-58.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-5801, SB-2405 |
| Status | Done |

## Why

A conversation started in a chat app can't be continued on the web.

## Acceptance criteria

- [x] A chat-app conversation shows in the person's Recent on the web with **Continue here**, from the traces they own
- [x] Each answer is checked again as them now; one they can no longer open every cite of shows as "no longer available to you"
- [x] Nobody else sees it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Conversations from traces
- [x] Web Recent and Continue here
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Agent.ListConversations` lists the caller's own conversations from their traces (`Agent.TracesOf`) and conversation records, always those started in a chat app; `Agent.OpenConversation` shows each turn checked again as them now (`turnAs`), one with a cite they can't open now `unavailable`. `GET /v1/me/conversations[/{id}]`; the web's **Your conversations** under Recent, fetched only when opened, with **Continue here**. Tested in TestYourConversations, TestCarryTheConversationThroughARunningCell and the web's carry test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
