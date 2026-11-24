---
title: "SB-3207: Notice to everyone"
summary: "There is no way to tell everyone the brain is down for an update."
---

# SB-3207: Notice to everyone

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.2 |
| Stories | US-32.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-09, LAW-11 |
| Depends on | — |
| Status | Done |

## Why

There is no way to tell everyone the brain is down for an update.

## Acceptance criteria

- [x] `PUT /v1/admin/notice` sets one plain line of at most 200 characters ending within 7 days; DELETE clears it; one record and an audit row
- [x] The web UI shows it on every screen and chat replies put it first until it ends
- [x] Too long or too far an end gets EINVAL; it ends by itself

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI and chat apps
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

ask.CellNotice in the traces store (cell_docs kind notice, id cell; no new store), PUT and DELETE /v1/admin/notice for operators, validated by ask.CheckNotice and audited with the text's SHA-256 only, GET /v1/notice for anyone signed in, {} once ended. The Slack and Mattermost apps put Notice: first in every answer; the web UI's NoticeBanner reads it on load and every five minutes. Tests: ask/notice_test.go, api/notice_test.go, chat apps' notice_test.go, tests/cell/notice_test.go, web notice.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
