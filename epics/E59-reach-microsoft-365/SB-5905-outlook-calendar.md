---
title: "SB-5905: Outlook calendar for meeting prep"
summary: "Meeting prep reads Google Calendar only."
---

# SB-5905: Outlook calendar for meeting prep

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.2 |
| Stories | US-59.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-30 |
| Depends on | SB-4202, SB-5903 |
| Status | Open |

## Why

Meeting prep reads Google Calendar only.

## Acceptance criteria

- [ ] Meeting prep, `/v1/me/events` and the leader's week read Outlook calendars through Graph
- [ ] Only the person's own events, with attendees resolved through Entra
- [ ] Another person's calendar is never read

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Outlook calendar connector
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
