---
title: "SB-4703: Agent spend caps"
summary: "One busy agent can spend a month's budget before anyone sees it."
---

# SB-4703: Agent spend caps

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.1 |
| Stories | US-47.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-07 |
| Depends on | SB-4105, SB-4701 |
| Status | Open |

## Why

One busy agent can spend a month's budget before anyone sees it.

## Acceptance criteria

- [ ] sb-tenant(5) `agents: { NAME: { monthly_usd } }`
- [ ] At 80% the owner is told; at 100% scheduled, triggered and shared runs pause until next month
- [ ] On-demand asks keep working under each space's budget

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest
- [ ] Guard
- [ ] Notify
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
