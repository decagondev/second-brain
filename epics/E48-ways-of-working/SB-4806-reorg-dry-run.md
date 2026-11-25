---
title: "SB-4806: Reorg dry run"
summary: "A move in the tree changes inheritance, escalation and agents, and nobody sees it before it happens."
---

# SB-4806: Reorg dry run

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.2 |
| Stories | US-48.2 |
| Manual | [`sb-space(1)`](../../manual/sb-space.1.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-11, LAW-24 |
| Depends on | SB-3902, SB-3907 |
| Status | Open |

## Why

A move in the tree changes inheritance, escalation and agents, and nobody sees it before it happens.

## Acceptance criteria

- [ ] `sb tree move --plan` and `sb tree merge --plan` list inherited settings, escalation paths, roll-ups, agents, routines, waits, rhythms and lost ownership
- [ ] The plan matches the tree after the manifest diff is applied in the test cell; nothing changes on `--plan`
- [ ] A move that would widen any access is refused with the path

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Tree diff
- [ ] CLI
- [ ] Manifest diff proposal
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
