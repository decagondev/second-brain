---
title: "SB-1504: The jobs service: every schedule, run"
summary: "The schedules and their watchdog are defined; nothing runs them."
---

# SB-1504: The `jobs` service: every schedule, run

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.2 |
| Stories | US-15.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-11, LAW-13, LAW-16, LAW-17 |
| Depends on | [SB-1501](SB-1501-cell-configuration-and-wiring.md) |
| Status | Done |

## Why

The schedules and their watchdog are defined; nothing runs them.

## Acceptance criteria

- [x] `cmd/sb-jobs` runs `sync`, `anchor`, `retention` (with legal holds and per-space days), `backup`, `report`, `access`, `review` and the watchdog, triggered by a local clock or Pub/Sub
- [x] Every run records its finish; the watchdog alerts on a missed run (sb-runtime(7), Schedules)
- [x] Legal holds and reporting lines are wired here and in `api`, closing PT-0011

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Job runner and triggers
- [x] Holds and reporting lines wired
- [x] Tests on a fake clock
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.2): `internal/cell/jobsvc` runs `sync`, `anchor`, `retention`, `backup`, `report`, `access` and `review`, each recording its finish (`state/runs/`) only when it succeeds, and the watchdog every 5 minutes over `schedule.Watch`. Triggers: the local clock (`Tick` each minute, with `schedule.Due`, a five-field cron matcher) and `sb-jobs run JOB` for Cloud Scheduler. `anchor` signs the last audit row with the cell's anchor key (`vault://tenants/TENANT/audit/anchor-seed` locally) and writes it write-once for seven years; `backup` refuses fewer than three copies on two media with one off-site (sb-cell(5), `backups`); `report` writes last week's report from a usage ledger (`internal/adapters/usage/localfs`) that also feeds the cost statement and the budget guard, now wired into the gateway; `access` checks a tenant-custody cell's paths; `review` opens the quarter's reviews. Legal holds and reporting lines are wired once in `internal/cell/wire` for every service, closing PT-0011 (BUG-0023). Tested on a fake clock in `tests/cell`: a day and the 1st of a quarter run every job at its minute; a failed backup records nothing and the watchdog alerts it as Sev-1; a new cell has every job overdue; a held trace survives retention and goes once released; three copies are written and two are refused. The `review` job was tested with no restricted space; the open-review path itself is SB-1202's law test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
