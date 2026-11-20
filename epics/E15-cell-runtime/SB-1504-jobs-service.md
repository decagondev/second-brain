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
| Status | Open |

## Why

The schedules and their watchdog are defined; nothing runs them.

## Acceptance criteria

- [ ] `cmd/sb-jobs` runs `sync`, `anchor`, `retention` (with legal holds and per-space days), `backup`, `report`, `access`, `review` and the watchdog, triggered by a local clock or Pub/Sub
- [ ] Every run records its finish; the watchdog alerts on a missed run (sb-runtime(7), Schedules)
- [ ] Legal holds and reporting lines are wired here and in `api`, closing PT-0011

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Job runner and triggers
- [ ] Holds and reporting lines wired
- [ ] Tests on a fake clock
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
