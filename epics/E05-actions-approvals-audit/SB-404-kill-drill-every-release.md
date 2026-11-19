---
title: "SB-404: Kill drill every release"
summary: "A switch that's never pulled is a switch that doesn't work."
---

# SB-404: Kill drill every release

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.1 Audit and kill |
| Stories | US-05.3 |
| Manual | [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md) |
| Laws | [LAW-10](../../manual/sb-laws.7.md#law-10) |
| Depends on | [SB-403](../E05-actions-approvals-audit/SB-403-kill-switches-per-agent-connector-and-tenant.md) |
| Status | In review |

## Why

A switch that's never pulled is a switch that doesn't work.

## Acceptance criteria

- [ ] Release pipeline fires each kill switch in staging and asserts effect (drill built; the pipeline step comes with the release controller, W5)
- [x] Drill results in release notes (`DrillReport` Markdown table)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Drill script
- [ ] Pipeline step (W5 release controller, staging)
- [x] Report
- [x] Write or extend the law tests: LAW-10
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.1b): `kill.Drill` fires each scope through the audited `Control`, polls the cell every 250 ms until it refuses, fails anything slower than 10 seconds, then clears the switch. `DrillReport` is the release-notes table. Tests drive a real checker on a fake clock: a live cell passes with every scope under 2.25 seconds, a frozen cell fails, and an empty drill never passes.

**Left:** the release pipeline step against a staging cell (W5).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
