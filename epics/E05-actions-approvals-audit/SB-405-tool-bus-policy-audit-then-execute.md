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
| Status | Open |

## Why

Every tool call takes the same path, so the laws hold for all of them at once.

## Acceptance criteria

- [ ] No tool can be invoked except through the bus (lint plus test)
- [ ] Order is fixed: kill check, authorise, audit row, approval check, execute

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Bus implementation
- [ ] Tool registry from release bundle
- [ ] Ordering test
- [ ] Write or extend the law tests: LAW-04, LAW-10
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
