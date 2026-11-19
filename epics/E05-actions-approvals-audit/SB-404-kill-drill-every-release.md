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
| Status | Open |

## Why

A switch that's never pulled is a switch that doesn't work.

## Acceptance criteria

- [ ] Release pipeline fires each kill switch in staging and asserts effect
- [ ] Drill results in release notes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Drill script
- [ ] Pipeline step
- [ ] Report
- [ ] Write or extend the law tests: LAW-10
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
