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
| Status | Open |

## Why

An operator must be able to stop one department's brain without stopping the company's.

## Acceptance criteria

- [ ] Every audit row carries its space; `sb audit tail --space` filters
- [ ] `sb kill space NAME` stops asks and actions in that space within seconds; company-wide asks drop its content
- [ ] Kill drill covers spaces

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Audit field
- [ ] Kill scope
- [ ] Drill update
- [ ] LAW-10 test for a space
- [ ] Write or extend the law tests: LAW-10, LAW-11
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
