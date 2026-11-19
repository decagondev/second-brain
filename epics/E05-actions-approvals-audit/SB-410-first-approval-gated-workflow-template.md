---
title: "SB-410: First approval-gated workflow template"
summary: "Each company picks one write workflow; the template makes that a manifest line."
---

# SB-410: First approval-gated workflow template

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.3 Execution |
| Stories | US-05.2 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-05](../../manual/sb-laws.7.md#law-05), [LAW-15](../../manual/sb-laws.7.md#law-15) |
| Depends on | [SB-408](../E05-actions-approvals-audit/SB-408-action-executor-with-receipts.md) |
| Status | Open |

## Why

Each company picks one write workflow; the template makes that a manifest line.

## Acceptance criteria

- [ ] `workflows: [ticket-update]` in the manifest enables it after that tenant's evals pass
- [ ] End-to-end test from draft to receipt

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Workflow definition
- [ ] Manifest field
- [ ] Eval precondition
- [ ] E2E test
- [ ] Write or extend the law tests: LAW-04, LAW-05, LAW-15
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
