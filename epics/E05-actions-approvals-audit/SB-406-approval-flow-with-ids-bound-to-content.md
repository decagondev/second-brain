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
| Status | In review |

## Why

An approval approves exactly one payload; any edit needs a new approval.

## Acceptance criteria

- [x] States: Drafted, PendingApproval, Approved, Rejected, Expired, Executing, Ran, DidNotRun
- [x] Approval id is bound to the SHA-256 of the canonical action payload; mismatch refuses
- [x] Approver cannot be the requester for consequential actions

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] State machine identical to `Approvals.tla`
- [x] Canonical payload encoding
- [x] Separation-of-duties rule
- [x] Expiry
- [x] LAW-15 property test
- [x] Write or extend the law tests: LAW-04, LAW-15
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.2a): `internal/core/approvals` holds the eight states and transitions of `Approvals.tla`, RFC 8785 canonical payloads (integers within ±2^53 only, UTF-16 key order), and a hash over payload, target and workflow. The bus checks approval in order: state, expiry (24 hours, then `Expired`), echoed hash, not the requester, policy, kill. The approval row is audited first, and the store moves state by compare-and-set. An edit voids any approval and re-proposes with the new hash. `TestLAW15_ApprovalBindsOnePayload` went red first, then green; it covers stale, made-up and self approvals and edits, and a mutation that ignores the echoed hash is caught. There is a memory action store.

**Left:** a Postgres action store (cell runtime) and a second reviewer for the approval rules.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
