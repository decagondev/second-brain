---
title: "SB-2906: Prompt growth watch"
summary: "Prompts can grow week by week and nobody sees it."
---

# SB-2906: Prompt growth watch

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.2 |
| Stories | US-29.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Open |

## Why

Prompts can grow week by week and nobody sees it.

## Acceptance criteria

- [ ] Health and the weekly report show the week's 95th percentile of input tokens per model call
- [ ] A warning when it is over 1.5 times the week before; counts only
- [ ] A doubled week warns; a flat one doesn't

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
