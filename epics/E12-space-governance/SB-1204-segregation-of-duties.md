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
| Status | Open |

## Why

Finance controls need more than 'approver isn't requester'.

## Acceptance criteria

- [ ] A workflow can require two approvers from different groups above a threshold the manifest sets
- [ ] No one approves an action they requested, or one requested by someone who reports to them
- [ ] `Approvals.tla` extended and re-checked

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest fields
- [ ] Approval rules
- [ ] Model update
- [ ] Proofs
- [ ] Write or extend the law tests: LAW-15
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
