---
title: "SB-4705: Agent access review"
summary: "Agents keep triggers, tools and spaces long after anyone needs them."
---

# SB-4705: Agent access review

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-11, LAW-16 |
| Depends on | SB-3504, SB-4402 |
| Status | Open |

## Why

Agents keep triggers, tools and spaces long after anyone needs them.

## Acceptance criteria

- [ ] Every 90 days each agent gets a review of its spaces, triggers and tools with what it used
- [ ] Anything unused is proposed off as a manifest diff to the stewards; nothing changes without them
- [ ] The review and its outcome are audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Job
- [ ] Proposal
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
