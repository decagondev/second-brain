---
title: "SB-4701: Agent value"
summary: "We know what an agent costs but not whether anyone uses what it drafts."
---

# SB-4701: Agent value

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.1 |
| Stories | US-47.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4004, SB-4602 |
| Status | Open |

## Why

We know what an agent costs but not whether anyone uses what it drafts.

## Acceptance criteria

- [ ] The registry and `GET /v1/admin/agents/value` show per agent and space, for 12 weeks, drafts made, published as is or edited, discarded, change rounds, time to publish, helpful rate and cost per published draft
- [ ] Owners see their agents', admins all; a doubled discard rate tells the owner once
- [ ] Counts only, matching the audit log

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Counts
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
