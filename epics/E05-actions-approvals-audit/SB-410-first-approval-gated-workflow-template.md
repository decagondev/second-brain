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
| Status | In review |

## Why

Each company picks one write workflow; the template makes that a manifest line.

## Acceptance criteria

- [x] `workflows: [ticket-update]` in the manifest enables it after that tenant's evals pass
- [x] End-to-end test from draft to receipt

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Workflow definition
- [x] Manifest field
- [x] Eval precondition
- [x] E2E test
- [x] Write or extend the law tests: LAW-04, LAW-05, LAW-15
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.3b): `internal/core/workflow` holds the `ticket-update` template: tool `ticket-update.comment`, connector `tickets`, consequential, with a strict payload of `{"comment"}` only, 1 to 10,000 characters. `Registry` builds the bus's tools from a manifest's workflows and approval groups. The Linear connector now writes: `Act` posts one comment whose id is a UUID derived from the idempotency key, reads it back when the answer has no id, fails rather than claiming a receipt, and implements `Find` for the reconciler. The recorded fake gained `CommentCreate` and `CommentByID` with lost-answer, no-id and no-op faults. `sb tenant apply` refuses, before any step, a plan that adds or changes a workflow without a passing eval record no more than 30 days old (`EEVAL`). The local control plane reads `evals/TENANT/WORKFLOW.json`, which `python3 -m evals run … --record FILE` writes, failures included. `TestTicketUpdateEndToEnd` runs the example manifest through the registry, the real cedar base policies, the bus and the Linear fake: propose over the API, an extra payload field refused, an approver outside `it-leads` refused, approval by `it-leads`, one comment at Linear, `Ran` with `linear:comment/…`, and the audit trail `PendingApproval Approved Executing Ran`. `TestLAW05_LinearActNeedsReceipt` checks the connector never claims a receipt it doesn't have.

**Left:** a run against a real Linear sandbox with a `comments:create` key, and Postgres stores (cell runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
