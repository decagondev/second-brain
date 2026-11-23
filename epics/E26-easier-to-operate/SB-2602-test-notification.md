---
title: "SB-2602: Send me a test"
summary: "Nobody can check a notification arrives until one silently doesn't."
---

# SB-2602: Send me a test

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.1 |
| Stories | US-26.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Done |

## Why

Nobody can check a notification arrives until one silently doesn't.

## Acceptance criteria

- [x] Send me a test in notification settings, and `sb selfhost check-notify --to EMAIL`, send one fixed message through the cell's real channel
- [x] At most once a minute per person; the result says whether the channel took it
- [x] The fake Slack, Mattermost and SMTP servers each receive exactly the fixed text

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API and CLI
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S25.1: POST /v1/me/notifications/test, POST /v1/admin/notifications/test, sb selfhost check-notify, Send me a test

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
