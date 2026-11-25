---
title: "SB-4702: Pause from the message"
summary: "Stopping a misbehaving agent in one space means finding an admin."
---

# SB-4702: Pause from the message

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.1 |
| Stories | US-47.1 |
| Manual | [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-10, LAW-11 |
| Depends on | SB-4402 |
| Status | Open |

## Why

Stopping a misbehaving agent in one space means finding an admin.

## Acceptance criteria

- [ ] Every agent message and draft carries **Pause here** for the space's stewards and the agent's owner
- [ ] The agent stops in that space at once, the reason goes to the owner, and it's audited
- [ ] A steward or the owner resumes it; waiting drafts are kept

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Kill switch scope
- [ ] Chat and UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
