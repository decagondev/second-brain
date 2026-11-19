---
title: "SB-604: `sb-onboarding(7)` day 0 to 90 runbook"
summary: "The runbook is the product for portfolio IT."
---

# SB-604: `sb-onboarding(7)` day 0 to 90 runbook

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.2 Onboarding |
| Stories | US-07.1 |
| Manual | [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md) |
| Laws | — |
| Depends on | [SB-601](../E07-rollout-ops/SB-601-sb-tenant-verify-on-real-data.md) |
| Status | In review |

## Why

The runbook is the product for portfolio IT.

## Acceptance criteria

- [x] Every step is a command or a console action with an expected output
- [x] Checklists for each phase

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Write runbook
- [ ] Dry-run with internal tenant (human; needs a deployed internal cell)
- [ ] Fix every confusing step (after the dry run)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.2): every step of sb-onboarding(7) now ends with what to expect, and each phase ends with its gate and a checklist. Steps name the real commands built so far: `sb tenant check` and `apply`, `sb tenant verify`, `python3 -m evals.pilot report`, `evals run --record`, `sb kill` and `unkill`, the approver view, and the usage API. `tools/check/runbook.py` (in `make examples`, with a self-test) fails the build if a step has no expected result or a phase has no gate or checklist.

**Left:** the dry run with the internal tenant, and fixing what it finds (human, deployed cell).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
