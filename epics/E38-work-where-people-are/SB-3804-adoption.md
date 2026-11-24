---
title: "SB-3804: Adoption"
summary: "The weekly report doesn't say how many people use the brain."
---

# SB-3804: Adoption

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.2 |
| Stories | US-38.2 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | SB-1607 |
| Status | Done |

## Why

The weekly report doesn't say how many people use the brain.

## Acceptance criteria

- [x] The weekly report adds N of M seats asked this week, with last week's
- [x] By space, people who asked, for spaces with at least 5
- [x] No one is named

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Report
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `report.Adoption`, `report.AdoptionOf`: the distinct people who asked this week and the week before, of the seats, and by space where at least 5 people asked (`ask.MinUsage`); `Report.Adoption` (`adoption` in `GET /v1/admin/usage`) and the report's **Adoption** row. Counts only. TestAdoption.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
