---
title: "SB-2903: Scope report"
summary: "A scope pattern that matches nothing, or far too much, goes unnoticed."
---

# SB-2903: Scope report

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.1 |
| Stories | US-29.2 |
| Manual | [`sb-space(1)`](../../manual/sb-space.1.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-16 |
| Depends on | — |
| Status | Done |

## Why

A scope pattern that matches nothing, or far too much, goes unnoticed.

## Acceptance criteria

- [x] `sb space scope-report` prints how many documents each space's scope patterns match now
- [x] It names patterns that match none and patterns that match more than half a connector's documents; it changes nothing
- [x] A typo is named; `/**` is named as too wide

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CLI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S28.1; tested in internal/core/ask/focus_test.go, internal/core/route/report_test.go, internal/cli/scopereport_test.go, tests/cell/retire_test.go and web/src/focus.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
