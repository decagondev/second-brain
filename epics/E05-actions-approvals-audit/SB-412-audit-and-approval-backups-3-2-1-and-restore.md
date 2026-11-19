---
title: "SB-412: Audit and approval backups (3-2-1) and restore drill"
summary: "The audit log is the one thing we can never lose."
---

# SB-412: Audit and approval backups (3-2-1) and restore drill

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.4 Hygiene |
| Stories | US-05.4 |
| Manual | [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11) |
| Depends on | [SB-401](../E05-actions-approvals-audit/SB-401-append-only-hash-chained-audit-log-written.md) |
| Status | Open |

## Why

The audit log is the one thing we can never lose.

## Acceptance criteria

- [ ] Three copies, two media, one off-site, with WORM retention
- [ ] Monthly restore drill verifies the chain end to end

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Backup jobs
- [ ] WORM bucket
- [ ] Restore drill script
- [ ] Drill schedule
- [ ] Write or extend the law tests: LAW-11
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
