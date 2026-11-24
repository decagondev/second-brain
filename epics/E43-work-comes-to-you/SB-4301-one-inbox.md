---
title: "SB-4301: One inbox"
summary: "What waits on a person is spread over tabs in every space, so things are missed."
---

# SB-4301: One inbox

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.1 |
| Stories | US-43.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-07, LAW-26 |
| Depends on | SB-4104 |
| Status | Open |

## Why

What waits on a person is spread over tabs in every space, so things are missed.

## Acceptance criteria

- [ ] `GET /v1/me/inbox` and the **Inbox** tab list, across spaces, what waits on me under Needs you, Failed and Done
- [ ] Each item is checked as me when shown (LAW-26); notifications carry counts only
- [ ] Each item links to where it's acted on; the inbox itself doesn't act

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
