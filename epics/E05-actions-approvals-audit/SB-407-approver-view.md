---
title: "SB-407: Approver view"
summary: "Approvers see what will run, where, and what changes."
---

# SB-407: Approver view

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.2 Approvals |
| Stories | US-05.2 |
| Manual | [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | [LAW-15](../../manual/sb-laws.7.md#law-15) |
| Depends on | [SB-406](../E05-actions-approvals-audit/SB-406-approval-flow-with-ids-bound-to-content.md) |
| Status | Open |

## Why

Approvers see what will run, where, and what changes.

## Acceptance criteria

- [ ] Shows target system, exact payload diff, requester, policy reason and expiry
- [ ] Approve and reject both require a reason field (optional text, required click)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI
- [ ] Diff renderer
- [ ] Notification hooks
- [ ] Write or extend the law tests: LAW-15
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
