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
| Status | In review |

## Why

The audit log is the one thing we can never lose.

## Acceptance criteria

- [ ] Three copies, two media, one off-site, with WORM retention (enforced by the job; local write-once targets built; a cloud bucket with object lock comes with W7 GCP)
- [x] Monthly restore drill verifies the chain end to end

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Backup jobs
- [ ] WORM bucket (GCP bucket with retention lock, W7)
- [x] Restore drill script
- [ ] Drill schedule (monthly, with the cell runtime's scheduler)
- [x] Write or extend the law tests: LAW-11
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.4b): `internal/core/backup`. `Check` enforces three copies, two media and one off-site, and the job refuses to run otherwise. `Job.Run` writes the full signed audit export and a JSON-lines snapshot of the approval store, each with a `.sha256`, to every copy, named by tenant and day, with a 7-year retention date. `Drill` fetches the latest export from every copy, checks its sha256, verifies the chain with the pinned key, and fails a copy more than a day behind the live chain or copies that disagree. Its Markdown report names each copy. `internal/adapters/backup/localfs` is a write-once target: exclusive create, read-only files, refused overwrite, and no delete before the retention date. `tests/backup` (in `make smoke`) backs up to three real local targets, refuses a second run on the same night, and runs the drill: it passes, then fails on a tampered NAS copy and names it, and fails stale copies.

**Left:** a cloud WORM bucket (GCP, W7) and the monthly schedule (cell runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
