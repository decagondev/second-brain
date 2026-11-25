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
| Status | Open |

## Why

Agents start on a clock or a source change, not when a pull request opens, a ticket is filed or a meeting is booked.

## Acceptance criteria

- [ ] Triggers gain `on: pr-opened` (GitHub), `on: ticket-filed` (Linear, with labels) and `on: event-booked` (calendar)
- [ ] A trigger runs as its approver and drafts only; events come through the signed event intake
- [ ] More than 10 events an hour batch into one run

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Trigger kinds
- [ ] Event intake
- [ ] Batching
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
