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
| Status | Done |

## Why

What waits on a person is spread over tabs in every space, so things are missed.

## Acceptance criteria

- [x] `GET /v1/me/inbox` and the **Inbox** tab list, across spaces, what waits on me under Needs you, Failed and Done
- [x] Each item is checked as me when shown (LAW-26); notifications carry counts only
- [x] Each item links to where it's acted on; the inbox itself doesn't act

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Legwork.Inbox, GET /v1/me/inbox: steward queues (company, every space, the person's own), policies to read, approvals (toolbus.Bus.Pending, wire inboxApprovals), own action items whose transcript they can open, suggested fixes, research plans, failed and finished runs, each from the call its own tab makes, as the caller, now (LAW-26). The Questions tab is now Inbox (web Inbox.tsx), and ?space= opens the app in a space. Tests: TestWorkComesToYouThroughARunningCell, inbox.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
