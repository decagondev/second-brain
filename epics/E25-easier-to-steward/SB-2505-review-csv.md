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
| Status | Done |

## Why

The review list can't be taken to a meeting.

## Acceptance criteria

- [x] Download CSV writes exactly the rows on screen, made in the browser from the loaded list; no new endpoint
- [x] Documents the steward can't open stay counts with their owner; cells are quoted against formula injection
- [x] The CSV's rows equal the list's; a hidden document's title is never in it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CSV
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S24.2; see manual/sb-ask.3.md

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
