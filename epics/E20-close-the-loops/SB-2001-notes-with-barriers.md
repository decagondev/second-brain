---
title: "SB-2001: Notes in cells with information barriers"
summary: "In a cell with information barriers, the bus places a call's target before it runs, and a note that doesn't exist yet can't be placed, so notes are refused."
---

# SB-2001: Notes in cells with information barriers

| Field | Value |
|---|---|
| Epic | [E20 Close the Loops](README.md) |
| Wave / slice | W19 / S19.1 |
| Stories | US-20.1 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-04, LAW-22 |
| Depends on | — |
| Status | Open |

## Why

In a cell with information barriers, the bus places a call's target before it runs, and a note that doesn't exist yet can't be placed, so notes are refused.

## Acceptance criteria

- [ ] A `note-publish` target is placed by the space it is published in, which is known before the write
- [ ] A steward on one side of a barrier can publish a note in their space; the barrier still refuses a target in a space across it
- [ ] The W16 refusal becomes an allow-for-the-space and deny-across-the-barrier test

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Placement for notes
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
