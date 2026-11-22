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
| Status | Open |

## Why

The health panel knows jobs and connectors, not that the disk is full or the certificate expires on Friday.

## Acceptance criteria

- [ ] The hourly host step writes the box's facts (disk per volume, certificate days left, newest dump and off-box copy times) to the cell's state
- [ ] `GET /v1/admin/health` gains `box`, with the same verdicts as `--doctor`: warn at 80% disk, 21 days, 26 hours; fail at 90%, 7 days, 50 hours; facts older than two hours or unreadable are `unknown`, never green
- [ ] A failing box check raises an alert through the existing alert port once a day, and the console's health panel shows a Box section

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Facts file and reader
- [ ] Verdicts shared with SB-1801
- [ ] API and console section
- [ ] Alert
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
