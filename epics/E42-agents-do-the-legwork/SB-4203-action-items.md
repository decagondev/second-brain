---
title: "SB-4203: Action items from meetings"
summary: "What people agreed to do in a meeting is lost once the transcript is filed."
---

# SB-4203: Action items from meetings

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.1 |
| Stories | US-42.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-16 |
| Depends on | SB-4006 |
| Status | Done |

## Why

What people agreed to do in a meeting is lost once the transcript is filed.

## Acceptance criteria

- [x] A transcript's decisions draft gains action items, each with an owner matched to a tenant user and a due date if one was said
- [x] Nothing reaches an owner before the steward confirms; each owner gets one `action` message and can mark items done
- [x] An item past its date shows in the owner's inbox; an unmatched owner stays as written and gets no message

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Extraction
- [x] Records
- [x] Notify
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.ExtractActions reads Action:/TODO:/AI: lines without a model, DraftActions fills a decisions draft as the steward it ran as (Stewards.ActionsOf), Confirm records them (ActionStore in memory and Postgres), the notify job tells each matched owner once (action), /v1/me/actions and done, /v1/spaces/NAME/actions. Web: confirm on the steward screen, Your action items on the Legwork tab. Tests: TestExtractActions, TestMatchOwner, TestLegworkThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
