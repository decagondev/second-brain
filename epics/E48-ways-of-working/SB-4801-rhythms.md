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
| Status | Done |

## Why

Every recurring moment in the company needs a steward to remember the right step at the right time.

## Acceptance criteria

- [x] `rhythms:` in sb-tenant(5) holds at most 12 dated steps over at most 120 days, each a draft, routine, check-in or request with a named approver; the lint refuses a thirteenth step
- [x] A started rhythm shows on the space's home and each step lands in its approver's inbox; no step publishes or sends without them
- [x] LAW-29 is a law test and goes green; a rhythm whose subject leaves the space stops at once

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest and lint
- [x] Scheduler in `sb-jobs`
- [x] Home and inbox
- [x] Five built-in rhythms
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `rhythms:` and `onboarding:` on a space (sb-space(5)), `manifest.Rhythms` with the built-in four and `RhythmProblem`, the lint refusing a thirteenth step, a step past day 120, a step without an approver and an ask that acts or widens (TestRhythmsRefused). `ask.Rhythms` (`internal/core/ask/rhythms.go`) starts, approves, stops and lists runs, and `Tick` in the notify job offers due steps; each step is a `StewardQuestion` with a `rhythm` ref in its approver's queue (person queue or the space's), so it shows in the inbox. Each read checks the subject first (LAW-29, `laws/law29.go`). The space home in the web app is unchanged in this slice; rhythms are read over the API. Runs and rosters live in the trace store (kinds `rhythm`, `roster`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
