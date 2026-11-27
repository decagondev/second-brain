---
title: "SB-6003: Check the Microsoft 365 setup"
summary: "A Microsoft 365 setup can be wrong in ways only a real tenant shows, including a write permission the cell should never hold."
---

# SB-6003: Check the Microsoft 365 setup

| Field | Value |
|---|---|
| Epic | [E60 Proven for Real](README.md) |
| Wave / slice | W59 / S59.1 |
| Stories | US-60.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-03, LAW-16 |
| Depends on | SB-5901, SB-5902, SB-5907 |
| Status | Done |

## Why

A Microsoft 365 setup can be wrong in ways only a real tenant shows, including a write permission the cell should never hold.

## Acceptance criteria

- [x] `sb selfhost check-microsoft` checks the tenant and app registration and refuses any write permission, naming it
- [x] Mail permission must be limited by an application access policy to the shared mailboxes the manifest names; each named site and drive must answer
- [x] Unmapped Purview labels are listed by id and count; each failure names its fix in `docs/runbooks/microsoft.md`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Permission and access policy checks
- [x] `docs/runbooks/microsoft.md`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `sb selfhost check-microsoft` reads the app registration's application permissions through its service principal's app role assignments, refuses any that isn't on the read-only list (named), checks each named shared mailbox reads and a `--probe` mailbox is refused (no probe fails), each drive answers, and lists the tenant's sensitivity labels `labels.microsoft` doesn't map. Fixes in `docs/runbooks/microsoft.md`. Tested in TestCheckMicrosoft against the Graph fake's new service principal, drive root, inbox and label routes.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
