---
title: "SB-406: Approval flow with ids bound to content hashes"
summary: "An approval approves exactly one payload; any edit needs a new approval."
---

# SB-406: Approval flow with ids bound to content hashes

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.2 Approvals |
| Stories | US-05.2 |
| Manual | [`sb-approvals(7)`](../../manual/sb-approvals.7.md), [`sb-actions(3)`](../../manual/sb-actions.3.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-15](../../manual/sb-laws.7.md#law-15) |
| Depends on | [SB-405](../E05-actions-approvals-audit/SB-405-tool-bus-policy-audit-then-execute.md) |
| Status | Open |

## Why

An approval approves exactly one payload; any edit needs a new approval.

## Acceptance criteria

- [ ] States: Drafted, PendingApproval, Approved, Rejected, Expired, Executing, Ran, DidNotRun
- [ ] Approval id is bound to the SHA-256 of the canonical action payload; mismatch refuses
- [ ] Approver cannot be the requester for consequential actions

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] State machine identical to `Approvals.tla`
- [ ] Canonical payload encoding
- [ ] Separation-of-duties rule
- [ ] Expiry
- [ ] LAW-15 property test
- [ ] Write or extend the law tests: LAW-04, LAW-15
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
