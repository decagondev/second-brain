---
title: "SB-2907: Start here goes stale"
summary: "A space's reading list can point new members at gone or old documents."
---

# SB-2907: Start here goes stale

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.2 |
| Stories | US-29.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

A space's reading list can point new members at gone or old documents.

## Acceptance criteria

- [x] The review list shows `start_here` items that are gone, stale, or that more than a quarter of the space's members can't open, each with why
- [x] An item the steward can't open is a count only
- [x] A deleted item is listed; a fine one isn't

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Review section
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S28.2; tested in internal/core/ask/startstale_test.go, tests/spaces/leftovers_test.go, tests/cell/leftovers_test.go, internal/core/health/prompt_test.go, internal/core/report, internal/cli and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
