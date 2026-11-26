---
title: "SB-5406: How far each agent reaches"
summary: "Nobody can see how much an agent can read through its approvers, triggers and loans, or notice when that grows."
---

# SB-5406: How far each agent reaches

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.2 |
| Stories | US-54.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-01, LAW-31 |
| Depends on | SB-4707, SB-5303 |
| Status | Open |

## Why

Nobody can see how much an agent can read through its approvers, triggers and loans, or notice when that grows.

## Acceptance criteria

- [ ] `GET /v1/admin/agents/NAME/reach` and the registry show the spaces and number of documents an agent can read through its approvers, triggers, rhythms and open loans, with how many are labelled or held
- [ ] Reach is in the agent's record (SB-4707) with the change since last week
- [ ] Reach growing by more than a tenth in a week alerts the agent owner once

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Reach from a replay as each approver
- [ ] Registry and record lines
- [ ] Owner alert
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
