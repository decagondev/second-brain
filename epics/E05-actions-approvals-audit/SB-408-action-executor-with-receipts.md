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
| Status | In review |

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

## Progress

2026-11-19 (slice S4.2a, built with the bus): `Bus.Execute` re-checks the kill switch, re-hashes the payload against the approved hash, writes the `Executing` audit row, and calls `Act` with the approval id as idempotency key under a 30-second timeout. Only a non-empty receipt makes `Ran`. A missing receipt is `DidNotRun` (`ENORECEIPT`), and an error is `DidNotRun`, with control characters stripped from the reason before it is audited. `TestLAW05_RanMeansReceipt`, red since W0, is green on the real bus.

**Left:** the reconciler for timeouts with unknown effect (it asks the target by idempotency key, never re-executes), then the remaining checkboxes.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
