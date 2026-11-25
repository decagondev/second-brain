---
title: "SB-4504: Waiting on another department"
summary: "Work blocked on another department is invisible to both, and nobody above them hears until it's late."
---

# SB-4504: Waiting on another department

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.2 |
| Stories | US-45.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-24 |
| Depends on | SB-4104, SB-4101, SB-4203 |
| Status | Open |

## Why

Work blocked on another department is invisible to both, and nobody above them hears until it's late.

## Acceptance criteria

- [ ] A request, goal or action item can be marked waiting on another space with a reason and a date
- [ ] Both homes list it, checked as the reader, and roll-ups count cross-space waits
- [ ] A wait a working day past its date goes once to the nearest common parent's stewards

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Field and API
- [ ] Job
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
