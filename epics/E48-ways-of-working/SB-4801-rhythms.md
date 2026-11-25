---
title: "SB-4801: Rhythms"
summary: "Every recurring moment in the company needs a steward to remember the right step at the right time."
---

# SB-4801: Rhythms

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.1 |
| Stories | US-48.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-departments(7)`](../../manual/sb-departments.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-16, LAW-26, LAW-29 |
| Depends on | SB-4302, SB-4301, SB-4601 |
| Status | Open |

## Why

Every recurring moment in the company needs a steward to remember the right step at the right time.

## Acceptance criteria

- [ ] `rhythms:` in sb-tenant(5) holds at most 12 dated steps over at most 120 days, each a draft, routine, check-in or request with a named approver; the lint refuses a thirteenth step
- [ ] A started rhythm shows on the space's home and each step lands in its approver's inbox; no step publishes or sends without them
- [ ] LAW-29 is a law test and goes green; a rhythm whose subject leaves the space stops at once

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest and lint
- [ ] Scheduler in `sb-jobs`
- [ ] Home and inbox
- [ ] Five built-in rhythms
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
