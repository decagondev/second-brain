---
title: "SB-2505: Review list as CSV"
summary: "The review list can't be taken to a meeting."
---

# SB-2505: Review list as CSV

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.2 |
| Stories | US-25.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

The review list can't be taken to a meeting.

## Acceptance criteria

- [ ] Download CSV writes exactly the rows on screen, made in the browser from the loaded list; no new endpoint
- [ ] Documents the steward can't open stay counts with their owner; cells are quoted against formula injection
- [ ] The CSV's rows equal the list's; a hidden document's title is never in it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] CSV
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
