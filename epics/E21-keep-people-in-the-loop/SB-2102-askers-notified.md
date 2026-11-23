---
title: "SB-2102: Askers hear about replies"
summary: "An asker finds a steward's reply only by opening their questions."
---

# SB-2102: Askers hear about replies

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.1 |
| Stories | US-21.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-2101 |
| Status | Done |

## Why

An asker finds a steward's reply only by opening their questions.

## Acceptance criteria

- [x] The notify job tells an asker "Your question has a reply" with a link to their questions, when a steward replies or publishes a note
- [x] The message carries no reply text and no cite; the link opens the reply under the asker's access
- [x] Each reply is told once

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Reply events in the job
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: Done. The same job tells each asker "N of your questions has a reply" or "…was published as a note" (`StewardReply.NoteAt`, set on first publish only) with `/?tab=questions`, which the web UI now opens on. No reply text or cite in the message; each event once, since the job keeps its last run in `state/notify/last`. Covered by the same law and cell tests.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
