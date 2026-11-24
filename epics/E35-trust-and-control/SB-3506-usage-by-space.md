---
title: "SB-3506: Usage by space"
summary: "The weekly report doesn't break anything down by space."
---

# SB-3506: Usage by space

| Field | Value |
|---|---|
| Epic | [E35 Trust and Control](README.md) |
| Wave / slice | W34 / S34.2 |
| Stories | US-35.2 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-07, LAW-19 |
| Depends on | SB-3101 |
| Status | Done |

## Why

The weekly report doesn't break anything down by space.

## Acceptance criteria

- [x] The weekly report adds one row per space: asks, the share that found nothing, and the share rated helpful
- [x] A space with under 5 asks shows fewer than 5
- [x] The steward screen shows its own space's row only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Report
- [x] Steward screen
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S34.2 with tests; make check green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
