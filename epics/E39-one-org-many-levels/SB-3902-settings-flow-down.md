---
title: "SB-3902: Settings flow down"
summary: "Every team space has to repeat its department's glossary, start-here list and agents."
---

# SB-3902: Settings flow down

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.1 |
| Stories | US-39.1 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-24 |
| Depends on | SB-3901 |
| Status | Open |

## Why

Every team space has to repeat its department's glossary, start-here list and agents.

## Acceptance criteria

- [ ] A child without its own `glossary`, `start_here`, `review_after_days` or `agents` takes its parent's; the child's own term wins
- [ ] A child can only tighten `label` and `retention_days`
- [ ] A parent's stewards are stewards of each child for escalations and roll-ups, not for publishing its notes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest
- [ ] Spaces
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
