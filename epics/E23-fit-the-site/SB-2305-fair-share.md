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
| Status | Open |

## Why

One person asking twenty questions can hold up a one-CPU box.

## Acceptance criteria

- [ ] `asks_per_minute` (default 6) per person, counted in the cell; over it, `ERATE` with the seconds to wait; search isn't limited
- [ ] A burst of ten asks gets six answers and four `ERATE`s
- [ ] The weekly report counts refusals

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Limiter
- [ ] Manifest field
- [ ] Report count
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
