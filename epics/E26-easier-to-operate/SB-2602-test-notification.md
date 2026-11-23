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
| Status | Open |

## Why

Nobody can check a notification arrives until one silently doesn't.

## Acceptance criteria

- [ ] Send me a test in notification settings, and `sb selfhost check-notify --to EMAIL`, send one fixed message through the cell's real channel
- [ ] At most once a minute per person; the result says whether the channel took it
- [ ] The fake Slack, Mattermost and SMTP servers each receive exactly the fixed text

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API and CLI
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
