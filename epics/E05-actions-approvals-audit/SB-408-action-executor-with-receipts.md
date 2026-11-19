---
title: "SB-408: Action executor with receipts"
summary: "'Ran' means the target said so."
---

# SB-408: Action executor with receipts

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.3 Execution |
| Stories | US-05.5 |
| Manual | [`sb-actions(3)`](../../manual/sb-actions.3.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-05](../../manual/sb-laws.7.md#law-05) |
| Depends on | [SB-406](../E05-actions-approvals-audit/SB-406-approval-flow-with-ids-bound-to-content.md), [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | Open |

## Why

'Ran' means the target said so.

## Acceptance criteria

- [ ] Action is `Ran` only with a receipt from `act`; otherwise `DidNotRun` with the reason
- [ ] Idempotency key per approval id prevents double execution

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Executor
- [ ] Receipt validation
- [ ] Idempotency store
- [ ] LAW-05 green with no-receipt mock
- [ ] Write or extend the law tests: LAW-05
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
