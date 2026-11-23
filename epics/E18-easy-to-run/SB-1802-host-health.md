---
title: "SB-1802: Host health in the console"
summary: "The health panel knows jobs and connectors, not that the disk is full or the certificate expires on Friday."
---

# SB-1802: Host health in the console

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.1 |
| Stories | US-18.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-03 |
| Depends on | SB-1801 |
| Status | Done |

## Why

The health panel knows jobs and connectors, not that the disk is full or the certificate expires on Friday.

## Acceptance criteria

- [x] The hourly host step writes the box's facts (disk per volume, certificate days left, newest dump and off-box copy times) to the cell's state
- [x] `GET /v1/admin/health` gains `box`, with the same verdicts as `--doctor`: warn at 80% disk, 21 days, 26 hours; fail at 90%, 7 days, 50 hours; facts older than two hours or unreadable are `unknown`, never green
- [x] A failing box check raises an alert through the existing alert port once a day, and the console's health panel shows a Box section

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Facts file and reader
- [x] Verdicts shared with SB-1801
- [x] API and console section
- [x] Alert
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.1): the hourly host step (`update.sh`) records the facts first, in `state/host/facts.json`, before the updater runs, so a refused release still leaves them fresh. `GET /v1/admin/health` adds `box` (`health.BoxFrom`) on a cell whose state has `host/`: the same verdicts as the doctor, unknown with no facts, unreadable facts or facts over two hours old. The drift job (hourly) raises one Sev-2 `box_check` a day per failing check, claimed in the usage ledger like the budget warning. The console's health panel gains a Box section with each check and its fix. Tests: `TestHealthJudgesTheBox` on a running cell (no box on a managed cell, unknown with no facts, a full disk failing first, one alert a day, old facts unknown), `TestBoxFromNoFactsIsUnknown`, and the console test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
