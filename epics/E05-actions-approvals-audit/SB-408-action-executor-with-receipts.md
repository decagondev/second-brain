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

- [x] Action is `Ran` only with a receipt from `act`; otherwise `DidNotRun` with the reason
- [x] Idempotency key per approval id prevents double execution

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Executor
- [x] Receipt validation
- [x] Idempotency store
- [x] LAW-05 green with no-receipt mock
- [x] Write or extend the law tests: LAW-05
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.2a, built with the bus): `Bus.Execute` re-checks the kill switch, re-hashes the payload against the approved hash, writes the `Executing` audit row, and calls `Act` with the approval id as idempotency key under a 30-second timeout. Only a non-empty receipt makes `Ran`. A missing receipt is `DidNotRun` (`ENORECEIPT`), and an error is `DidNotRun`, with control characters stripped from the reason before it is audited. `TestLAW05_RanMeansReceipt`, red since W0, is green on the real bus.

2026-11-19 (slice S4.3a): the idempotency store (`toolbus.Receipts`; memory adapter) keeps every receipt under its key, and the executor checks it before `Act`, so a key that already ran returns its receipt without a second call. An `Act` error or timeout leaves the action `DidNotRun` and `unconfirmed`, while a no-receipt answer is confirmed. `Bus.Reconcile` (every 5 minutes through `RunReconciler`) asks connectors that implement `toolbus.Finder`. When it finds the change, it writes a `Ran` audit row with the receipt (actor `sys_reconciler`), stores the receipt, and records it on the action, without ever calling `Act`. After 24 hours with nothing found it gives up. The manual gained the exact rules in sb-approvals(7).

**Left:** Postgres stores for actions and receipts (cell runtime), and `Find` on real connectors that write (SB-410's Linear write).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
