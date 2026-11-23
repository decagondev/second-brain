---
title: "SB-2607: Notifications in health"
summary: "Nobody sees that notifications stopped going out."
---

# SB-2607: Notifications in health

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.2 |
| Stories | US-26.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Open |

## Why

Nobody sees that notifications stopped going out.

## Acceptance criteria

- [ ] The notify job keeps its last run's counts (sent, skipped, failed) per channel, shown in `GET /v1/admin/health` and one line of the weekly report
- [ ] A warning when the last run failed or hasn't run for an hour; counts never name anyone
- [ ] A failing fake channel turns health to warning

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Counts
- [ ] Health and report
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
