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
| Status | Open |

## Why

Operators hear about spend only when answers stop, and the weekly report is only a file.

## Acceptance criteria

- [ ] A `budget_warning` alert once a month when spend passes 80% of `budget_usd_month`, before `EBUDGET`
- [ ] The weekly report is posted to the alert webhook or a report webhook as well as written to the state directory; a failed delivery is a failed run
- [ ] Tests on a fake clock

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Budget warning
- [ ] Report delivery
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
