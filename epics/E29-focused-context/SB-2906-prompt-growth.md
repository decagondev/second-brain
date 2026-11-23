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
| Status | Done |

## Why

Prompts can grow week by week and nobody sees it.

## Acceptance criteria

- [x] Health and the weekly report show the week's 95th percentile of input tokens per model call
- [x] A warning when it is over 1.5 times the week before; counts only
- [x] A doubled week warns; a flat one doesn't

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Counts
- [x] Health and report
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S28.2; tested in internal/core/ask/startstale_test.go, tests/spaces/leftovers_test.go, tests/cell/leftovers_test.go, internal/core/health/prompt_test.go, internal/core/report, internal/cli and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
