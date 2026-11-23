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
| Status | Open |

## Why

A scope pattern that matches nothing, or far too much, goes unnoticed.

## Acceptance criteria

- [ ] `sb space scope-report` prints how many documents each space's scope patterns match now
- [ ] It names patterns that match none and patterns that match more than half a connector's documents; it changes nothing
- [ ] A typo is named; `/**` is named as too wide

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] CLI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
