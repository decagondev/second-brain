---
title: "SB-1204: Segregation of duties for workflows"
summary: "Finance controls need more than 'approver isn't requester'."
---

# SB-1204: Segregation of duties for workflows

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.2 Barriers and duties |
| Stories | US-12.2 |
| Manual | [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | [LAW-15](../../manual/sb-laws.7.md#law-15) |
| Depends on | [SB-406](../E05-actions-approvals-audit/SB-406-approval-flow-with-ids-bound-to-content.md) |
| Status | Done |

## Why

Finance controls need more than 'approver isn't requester'.

## Acceptance criteria

- [x] A workflow can require two approvers from different groups above a threshold the manifest sets
- [x] No one approves an action they requested, or one requested by someone who reports to them
- [x] `Approvals.tla` extended and re-checked

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest fields
- [x] Approval rules
- [x] Model update
- [x] Proofs
- [x] Write or extend the law tests: LAW-15
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.2b): sb-approvals(7) gained "Segregation of duties, exactly", sb-tenant(5) the `duties` field and its rules, and sb-identity(7) the SCIM manager. The manifest's `duties: { WORKFLOW: { second: group:NAME, above: N } }` is checked (a known workflow, a different group, a whole number) and in the published schema; `workflow.Registry` puts it on the tool. The tool bus asks for two approvals when the payload's `amount` is above `N` or isn't a whole number: the first from the workflow's group (recorded as `first_approval`, audited, the action stays `PendingApproval`), the second from the duty's group by someone else on the same hash; an edit voids both. No approver may be the requester or anyone in the requester's reporting line (`approvals.ReportingLine` from the directory's `manager`, which SCIM now keeps); an unreadable or overlong line refuses. A new platform forbid refuses an approval by anyone outside the group named for it, because the new law test found that a tenant permit for IT leads let one of them give the finance approval. `Approvals.tla` gained the second approver and the reporting line, with `NoManagerApproval` and `TwoApprovers`; TLC is clean and the three new mutations are caught (19 of 19), and `crosscheck.py` mirrors it. `TestLAW15_SegregationOfDuties` drives the real bus and policies; a bus without reporting lines or without the duty is caught, and the test fails unless actions run after one and after two approvals.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
