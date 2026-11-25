---
title: "SB-5105: Branch moderators"
summary: "A department steward can pause only the agents of their own space, not the teams below it."
---

# SB-5105: Branch moderators

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.2 |
| Stories | US-51.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md) |
| Laws | LAW-10, LAW-24 |
| Depends on | SB-4702, SB-3901 |
| Status | Open |

## Why

A department steward can pause only the agents of their own space, not the teams below it.

## Acceptance criteria

- [ ] A steward of a space with children may pause any agent and review any shared skill, routine or rhythm in their branch
- [ ] A child they can't open shows names and counts, never content
- [ ] A sibling branch is refused, and every action is audited with the branch

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Branch scope for pause
- [ ] Branch review list
- [ ] Audit
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
