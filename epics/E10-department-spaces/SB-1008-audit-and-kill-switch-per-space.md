---
title: "SB-1008: Audit and kill switch per space"
summary: "An operator must be able to stop one department's brain without stopping the company's."
---

# SB-1008: Audit and kill switch per space

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.3 Ask in a space |
| Stories | US-10.4 |
| Manual | [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | [LAW-10](../../manual/sb-laws.7.md#law-10), [LAW-11](../../manual/sb-laws.7.md#law-11) |
| Depends on | [SB-1002](SB-1002-space-registry-and-space-scoped-request-context.md), [SB-403](../E05-actions-approvals-audit/SB-403-kill-switches-per-agent-connector-and-tenant.md) |
| Status | Done |

## Why

An operator must be able to stop one department's brain without stopping the company's.

## Acceptance criteria

- [x] Every audit row carries its space; `sb audit tail --space` filters
- [x] `sb kill space NAME` stops asks and actions in that space within seconds; company-wide asks drop its content
- [x] Kill drill covers spaces

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Audit field
- [x] Kill scope
- [x] Drill update
- [x] LAW-10 test for a space
- [x] Write or extend the law tests: LAW-10, LAW-11
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.3c): sb-kill-switches(7), sb-kill(1), sb-audit(5), sb-audit(7), sb-audit(1) and sb-admin(3) changed first. `space:NAME` is a kill scope: `kill.Op.Space` stops asks (the ask agent), drafts and tool calls (the tool bus, at proposal, approval and execution, from the action's recorded space) in that space, and `Checker.SpaceKilled` makes retrieval refuse every chunk a killed space holds (`space_killed`), even through another holder, so company-wide answers drop it; a killed tenant or an unloaded list counts as every space killed. `sb kill space NAME` and `sb unkill space NAME` work against the local control plane. Audit rows carry `space` (`spc_NAME`) for decisions, tool calls and approvals in a space; it is hashed, absent for company-wide and cell-level rows, so older rows hash exactly as before; `approvals.Action` records its space. `audit.InSpace` filters a space's rows and its kill rows; the admin audit API takes `space=` (`sb audit tail` itself still needs a running cell, SB-208). `kill.DrillScopes` adds every space to the release drill. Apply no longer refuses spaces: W10 is built, and the smoke test applies `spaces.yaml` and then finds no changes. Two law tests: `TestLAW10_SpaceKillStopsTheSpaceOnly` (real checker and retrieval on a fake clock; retrieval that ignores space kills is caught) and `TestLAW11_SpaceIsInTheChain` (moving a row between spaces breaks the chain; a hash without the space is caught).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
