---
title: "SB-2603: Check the chat bot"
summary: "A bot missing a scope or permission fails silently."
---

# SB-2603: Check the chat bot

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.1 |
| Stories | US-26.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | In review |

## Why

A bot missing a scope or permission fails silently.

## Acceptance criteria

- [x] `sb selfhost check-chat` asks Slack (`auth.test`, scopes) or Mattermost (`users/me`, team membership) what the bot may do, without printing the token
- [x] Each missing scope or permission is named with where it is granted
- [x] Fakes with each scope missing; a run against a real Mattermost is logged on this ticket

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CLI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S25.1: sb selfhost check-chat for Slack and Mattermost, tested against fakes with each scope or permission missing. The run against a real Mattermost is still to be logged here

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
