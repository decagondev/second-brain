---
title: "SB-2305: Fair share on a small box"
summary: "One person asking twenty questions can hold up a one-CPU box."
---

# SB-2305: Fair share on a small box

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.2 |
| Stories | US-23.3 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-errors(7)`](../../manual/sb-errors.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | — |
| Depends on | — |
| Status | Done |

## Why

One person asking twenty questions can hold up a one-CPU box.

## Acceptance criteria

- [x] `asks_per_minute` (default 6) per person, counted in the cell; over it, `ERATE` with the seconds to wait; search isn't limited
- [x] A burst of ten asks gets six answers and four `ERATE`s
- [x] The weekly report counts refusals

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Limiter
- [x] Manifest field
- [x] Report count
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: done in S22.2. `asks_per_minute` (default 6, 1 to 600) per person over a sliding minute, counted in the cell's memory (`ask.Rate`); one more is `ERATE` (429, `Retry-After` and `retry_after`) with no model call, traced as agent `limited` with no question. Search and the verifier's sampled asks aren't limited; the chat apps say how long to wait. A burst of ten gets six answers and four `ERATE`s (`TestAsksPerMinute`); the weekly report counts the refusals. Manuals: sb-tenant(5), sb-errors(7), sb-ask(3), sb-observability(7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
