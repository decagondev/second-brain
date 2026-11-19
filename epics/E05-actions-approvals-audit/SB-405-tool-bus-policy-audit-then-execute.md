---
title: "SB-405: Tool bus: policy, audit, then execute"
summary: "Every tool call takes the same path, so the laws hold for all of them at once."
---

# SB-405: Tool bus: policy, audit, then execute

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.2 Approvals |
| Stories | US-05.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-10](../../manual/sb-laws.7.md#law-10) |
| Depends on | [SB-401](../E05-actions-approvals-audit/SB-401-append-only-hash-chained-audit-log-written.md), [SB-109](../E02-policy/SB-109-policy-engine-v0-authorise-returns-allow.md) |
| Status | In review |

## Why

Every tool call takes the same path, so the laws hold for all of them at once.

## Acceptance criteria

- [x] No tool can be invoked except through the bus (lint plus test)
- [x] Order is fixed: kill check, authorise, audit row, approval check, execute

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Bus implementation
- [ ] Tool registry from release bundle (the bus takes a registry; loading it from the signed bundle is SB-503)
- [x] Ordering test
- [x] Write or extend the law tests: LAW-04, LAW-10
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.2a, with SB-406 and the executor of SB-408): `internal/core/toolbus`. A plain call goes through kill check, `call_tool`, an `Executing` audit row, `Act`, then `Ran` or `DidNotRun`. A consequential call goes through kill check, `propose`, and a `PendingApproval` audit row, then stops. An unregistered tool is `EPERM`, and an audit failure means nothing runs. Ordering tests assert the exact sequence (`audit:Executing act audit:Ran`; `audit:Approved audit:Executing act audit:Ran`). `tools/check/toolbus.py` (in `make lint`, with a self-test) fails any `.Act(` call outside the bus, the connector kit and the laws. Base policies gained tenant-scoped permits for `call_tool` (non-consequential tools), `propose`, and `approve` by the workflow's approver group. `TestLAW04_AuditBeforeAct`, red since W0, is green on the real bus, and a mutation whose audit rows are never stored is caught. `make proofs` and `make crosscheck` pass.

**Left:** loading the registry from the signed release bundle (SB-503).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
