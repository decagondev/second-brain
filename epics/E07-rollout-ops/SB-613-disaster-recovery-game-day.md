---
title: "SB-613: Disaster-recovery game day"
summary: "Restore a cell from backups inside its region, on the clock."
---

# SB-613: Disaster-recovery game day

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.4 Harden |
| Stories | US-07.6 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-412](../E05-actions-approvals-audit/SB-412-audit-and-approval-backups-3-2-1-and-restore.md), [SB-210](../E03-connectors-ingestion/SB-210-index-rebuild-from-sources.md) |
| Status | In review |

## Why

Restore a cell from backups inside its region, on the clock.

## Acceptance criteria

- [ ] Cell restored within RTO 4 hours, RPO 15 minutes (scored by `gameday.Score`; rehearsed on local adapters in `tests/gameday`; the real game day in staging is human-led, and the 15-minute RPO needs the cloud adapter's continuous log archive, W7)
- [ ] Audit chain verifies after restore (`TestLAW11_ChainSurvivesRestore` and the rehearsal; confirmed on the day in staging)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Plan
- [ ] Run (human, in staging: [runbook](../../docs/runbooks/dr-game-day.md))
- [ ] Retro and fixes (template in the runbook)
- [x] Write or extend the law tests: LAW-11, LAW-13
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.4a): sb-observability(7) gained "Disaster-recovery game day, exactly": the scenario, the targets, and a scoring table with RTO, RPO, chain, index and region. `internal/core/gameday.Restore` checks every copy as the restore drill does and restores from the first one that passes, naming the copies it skipped. It refuses when copies that pass disagree, writes only into empty stores, and adds rows one by one, refusing any that don't continue the chain. It restores the night's approvals snapshot too, and never restores the index. `gameday.Score` scores a day, and a day passes only if every line passes. `tests/gameday` (now part of `make check`) rehearses the whole scenario on local adapters: back up, lose the database and the index, restore, rebuild, verify, continue, score. It shows that a failure an hour after a nightly backup honestly fails the RPO. The new law test, `TestLAW11_ChainSurvivesRestore`, runs with zero to three tampered copies, and a self-test shows that a restore which trusts an unchecked copy is caught. LAW-13 is covered by the existing rebuild drill and by the rehearsal's index check. The runbook `docs/runbooks/dr-game-day.md` covers the plan, the run (each step with its expected result) and the retro.

**Left:** the game day itself, in staging, run by people, and the 15-minute RPO, which needs W7's continuous log archive.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
