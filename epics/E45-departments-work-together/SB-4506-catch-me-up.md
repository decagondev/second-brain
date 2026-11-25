---
title: "SB-4506: Catch me up"
summary: "Someone back from leave or new to a project reads weeks of changes to find what matters."
---

# SB-4506: Catch me up

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.2 |
| Stories | US-45.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-14, LAW-26 |
| Depends on | SB-4301, SB-4304, SB-2506 |
| Status | Done |

## Why

Someone back from leave or new to a project reads weeks of changes to find what matters.

## Acceptance criteria

- [x] **Catch me up** on a space, branch or collection since a date or the last visit, at most 90 days, gives one cited summary
- [x] A document the person has since lost is left out
- [x] It's offered once on the inbox after a notification pause ends

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] API
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `POST /v1/catch-up` (`Legwork.CatchUp`): the documents of the space and its branch scopes (or of a collection) with a version first seen since the day, at most 90 days back and 40 documents (`changedDocs`), kept only if the person can open them now, each checked in its own space's scope (`openDocs`), then one ask on the new `catch-up` legwork agent (prompt `catch-up`, base.cedar, agents.json, suite `agent-catch-up`) narrowed to them (`withCatchUp`). `catch_up` says since, documents and the person's Needs you items there. A pause carries `paused_at`; once it ends the inbox offers it once (`Prefs.CatchUpDue`, `Cell.catchUpOffer`, `Inbox.CatchUp`). The web UI remembers each space's last visit for the default day. TestCatchUpSince, TestCatchUpDueAfterAPause and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
