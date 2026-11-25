---
title: "SB-4707: An agent's record"
summary: "An auditor asking how an agent behaved gets a dozen screens."
---

# SB-4707: An agent's record

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-audit(1)`](../../manual/sb-audit.1.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4702, SB-4705 |
| Status | Open |

## Why

An auditor asking how an agent behaved gets a dozen screens.

## Acceptance criteria

- [ ] `sb agents record NAME` and **Download record** write one signed JSON file and its Markdown
- [ ] It holds the definition, owner, versions, eval scores, value, pauses, spend, access reviews and incidents
- [ ] No document text or asker names; the signature verifies

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] CLI
- [ ] API
- [ ] Signing
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
