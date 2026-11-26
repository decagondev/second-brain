---
title: "SB-5301: Start on a work event"
summary: "Agents start on a clock or a source change, not when a pull request opens, a ticket is filed or a meeting is booked."
---

# SB-5301: Start on a work event

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.1 |
| Stories | US-53.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-01, LAW-16, LAW-29 |
| Depends on | SB-4006, SB-4404 |
| Status | Done |

## Why

Agents start on a clock or a source change, not when a pull request opens, a ticket is filed or a meeting is booked.

## Acceptance criteria

- [x] Triggers gain `on: pr-opened` (GitHub), `on: ticket-filed` (Linear, with labels) and `on: event-booked` (calendar)
- [x] A trigger runs as its approver and drafts only; events come through the signed event intake
- [x] More than 10 events an hour batch into one run

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Trigger kinds
- [x] Event intake
- [x] Batching
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built for pull requests and tickets. sb-space(5) `on` takes `pr-opened` and `ticket-filed` (`manifest.SpaceTriggers`); `ask.IsWorkEvent` says a document's first chunk is a GitHub pull request (`code`, `/pulls/`) or a ticket; `Cell.workEvents` finds them first seen in the notify run's window in the spaces that watch them, and `notify.Job.workEvents` drafts each once ever (clock `work-…`), or one draft for more than `MaxWork` (10) of one space and kind; `Cell.runWork` drafts with `ask.WorkQuestion`, narrowed to the document, as the space's first active steward, kind `work-event`. A calendar trigger isn't built: meeting prep (SB-4202) already starts from booked meetings, and a calendar isn't an indexed source. The events come from the index, not the signed events out (SB-4404, which posts out). Tested in TestWorkEventsDraftOnceAndBatch and TestIsWorkEvent.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
