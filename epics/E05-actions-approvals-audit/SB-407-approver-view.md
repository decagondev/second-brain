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
| Status | In review |

## Why

Approvers see what will run, where, and what changes.

## Acceptance criteria

- [x] Shows target system, exact payload diff, requester, policy reason and expiry
- [x] Approve and reject both require a reason field (optional text, required click)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI
- [x] Diff renderer
- [x] Notification hooks (`toolbus.Bus.Notify`, ids only; the chat and email adapters come with the cell runtime)
- [x] Write or extend the law tests: LAW-15
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.2b): `internal/api/actions.go` serves `POST /v1/actions` (the registry's tool for the workflow; a disabled workflow is `EPERM`), `GET /v1/actions/{id}` (requester, approver group or auditor only, with `policy_reason` and a field-by-field `diff` when the connector implements `toolbus.Current`, else `null`), and approve and reject, with sb-errors(7) codes. API tests run the real bus over memory stores: propose, outsider read refused, approver read with diff, self-approval and wrong hash refused (409), approve to `Ran` with a receipt, reject, and float payloads refused. The web UI's approver view (`/?action=tc_…`) shows the target system and object, requester, policy reason, expiry, hash, exact payload and diff table. It echoes the hash it was shown, so the test changes the payload on the server after load and checks the stale hash is still what gets sent. Approve and Reject need explicit clicks (Enter does nothing), notes are optional, and the view is axe-clean. The bus records the policy reason on each action and calls `Notify` with ids only.

**Left:** the chat and email notification adapters (cell runtime), and a human WCAG pass with the web UI's (SB-307).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
