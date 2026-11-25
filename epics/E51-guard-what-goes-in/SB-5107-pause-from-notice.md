---
title: "SB-5107: Pause from the notice"
summary: "To pause an agent from its own notification, people have to find the agent first."
---

# SB-5107: Pause from the notice

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.2 |
| Stories | US-51.2 |
| Manual | [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-10 |
| Depends on | SB-4702 |
| Status | Open |

## Why

To pause an agent from its own notification, people have to find the agent first.

## Acceptance criteria

- [ ] Every notification about an agent run carries **Pause this agent here** for people allowed to pause it
- [ ] The link asks for a reason and pauses only that agent in that space, the same audited pause as SB-4702
- [ ] People not allowed to pause never see the link

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Link on agent notices
- [ ] Pause with reason
- [ ] Tests
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
