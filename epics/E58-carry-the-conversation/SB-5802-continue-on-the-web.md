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
| Status | Open |

## Why

A conversation started in a chat app can't be continued on the web.

## Acceptance criteria

- [ ] A chat-app conversation shows in the person's Recent on the web with **Continue here**, from the traces they own
- [ ] Each answer is checked again as them now; one they can no longer open every cite of shows as "no longer available to you"
- [ ] Nobody else sees it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Conversations from traces
- [ ] Web Recent and Continue here
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
