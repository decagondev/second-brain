---
title: "SB-4202: Meeting prep"
summary: "People walk into meetings without the documents, decisions and open questions the brain already holds."
---

# SB-4202: Meeting prep

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.1 |
| Stories | US-42.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-09, LAW-12 |
| Depends on | SB-3906 |
| Status | Done |

## Why

People walk into meetings without the documents, decisions and open questions the brain already holds.

## Acceptance criteria

- [x] **Prepare** takes a pasted agenda or an event from a read-only calendar connector, read as the person
- [x] The brief lists related documents, decisions of the last 90 days, open questions and what each tenant attendee owns, each checked as the person
- [x] Agenda text is data (LAW-09); a deprovisioned attendee isn't shown as an owner

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Calendar connector (read-only, recorded fake)
- [x] Agent and prompt
- [x] API
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Legwork Prepare/Events/PrepDue, the meeting-prep agent and prompt, internal/adapters/connectors/gcal (read-only, as the person, through gdrive.ServiceAccount with the calendar scope; recorded fake gcal/caltest), sb-cell(5) calendar, notify prep before meetings with prefs prep. Attendees known only as one active user (LAW-12). Tests: TestAgendaItems, the gcal tests, TestLegworkThroughARunningCell, agent-meeting-prep suite (with an injected agenda item).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
