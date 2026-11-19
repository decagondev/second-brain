---
title: "DR game day runbook"
summary: "Lose a cell's database in staging, restore it from backups inside its region, and score it (SB-613)."
---

# DR game day runbook

Quarterly, in staging, on a cell holding only test tenants (`acme-logistics`, `beta-freight`). The scenario, the targets and the scoring are in [sb-observability(7), Disaster-recovery game day, exactly](../../manual/sb-observability.7.md#disaster-recovery-game-day-exactly-sb-613). Rehearse first on local adapters: `go test ./tests/gameday/...` must pass.

## Roles

| Role | Does |
|---|---|
| Lead | runs the steps and keeps the clock |
| Scribe | writes every timestamp (UTC) in the retro as it happens |
| Observer | a second engineer who checks each expected result |

## Plan (a week before)

1. Book a two-hour window and tell the staging users. Expect: an announcement in the team channel with the window and the cell's tenant.
2. Run the restore drill (`sb audit verify --export` on each copy, SB-412). Expect: every copy passes, so a failure on the day is the scenario and not an old fault.
3. Record the cell's pinned audit key and its pinned region from its custody record. Expect: both are written in the retro before the day.

## Run

1. Snapshot the index's `(cite, ACL)` set (the rebuild drill's snapshot, SB-210). Expect: a non-empty set, saved outside the cell.
2. Declare the failure and write T0. Expect: T0 is in the retro to the second.
3. Stop the cell and drop its database (staging only). Expect: the cell's health check fails and asks return `EUNAVAIL`.
4. Provision an empty database **in the pinned region**. Expect: the region matches the custody record. Any other region stops the game day as a failure.
5. Restore with `gameday.Restore` from the 3-2-1 copies into the empty stores. Expect: it names the copy it used and the copies it skipped, and the number of rows and actions restored.
6. Verify the restored chain with the pinned key (`sb audit verify --export`). Expect: it verifies, and its last seq is the backup's.
7. Rebuild the index from the sources (SB-210). Expect: the `(cite, ACL)` set equals step 1's snapshot.
8. Start the cell and run its smoke test. Expect: the smoke test passes. Write this time as "answered".
9. Write one audit row (an operator `config` proposal) and verify again. Expect: the chain continues from the restored last row.
10. Score the day with `gameday.Score` and paste its Markdown into the retro. Expect: one line each for RTO, RPO, chain, index and region, each pass or FAIL.

## Retro (the same day)

Copy into the devlog. Use ids, counts and times only.

| Item | Value |
|---|---|
| Cell | `acme-logistics` (staging) |
| T0 | YYYY-MM-DDTHH:MM:SSZ |
| Answered | YYYY-MM-DDTHH:MM:SSZ |
| RTO (≤ 4 h) | |
| Last restored row | YYYY-MM-DDTHH:MM:SSZ |
| RPO (≤ 15 min) | |
| Copy used, copies skipped | |
| Chain verifies and continues | yes / no |
| Index equals the snapshot | yes / no |
| Region | yes / no |

**What went wrong, and the test that would have caught it:** each one goes in `living/BUGS_AND_MITIGATIONS.md`.

**Fixes before the next game day:** owner and ticket for each.

## After

A failed line is a finding: log it, fix it, and repeat the scenario at the next game day. Laws: [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-13](../../manual/sb-laws.7.md#law-13).
