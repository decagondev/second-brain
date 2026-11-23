---
title: "SB-2201: Orphaned documents"
summary: "When someone leaves, their documents keep answering with no owner to ask."
---

# SB-2201: Orphaned documents

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.1 |
| Stories | US-22.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-12, LAW-01 |
| Depends on | — |
| Status | Open |

## Why

When someone leaves, their documents keep answering with no owner to ask.

## Acceptance criteria

- [ ] The stewards' review list gains documents whose source owner is no active directory user, each checked live as the steward and shown as "owner gone"; those the steward can't open are counted
- [ ] A deprovisioned owner is never shown as someone to ask
- [ ] Deprovision an owner, and their document moves to the list within the next sync

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Owner check
- [ ] Review list kind
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
