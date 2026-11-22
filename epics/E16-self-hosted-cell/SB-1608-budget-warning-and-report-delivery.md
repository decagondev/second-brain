---
title: "SB-1608: Budget warning at 80% and the weekly report delivered"
summary: "Operators hear about spend only when answers stop, and the weekly report is only a file."
---

# SB-1608: Budget warning at 80% and the weekly report delivered

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.4 |
| Stories | US-16.1 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-16 |
| Depends on | — |
| Status | Done |

## Why

Operators hear about spend only when answers stop, and the weekly report is only a file.

## Acceptance criteria

- [x] A `budget_warning` alert once a month when spend passes 80% of `budget_usd_month`, before `EBUDGET`
- [x] The weekly report is posted to the alert webhook or a report webhook as well as written to the state directory; a failed delivery is a failed run
- [x] Tests on a fake clock

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Budget warning
- [x] Report delivery
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S15.4): the budget guard sends one `budget_warning` (Sev-3) the first time a metered call takes the month's spend past 80% of `budget_usd_month`, while answers still work; an exclusive-create marker in the usage ledger makes it once a month across every service and restart, and a budget of 0 sends none (`TestBudgetWarningOnceAMonth` on a fake clock, `TestFirstClaimWins`). The `report` job now posts the weekly report as JSON to `alerts.report_webhook`, or `alerts.webhook` when that is the only one set, as well as writing the file; a delivery that fails after its retry fails the run, so the watchdog sees it (`TestWeeklyReportIsDelivered`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
