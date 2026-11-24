---
title: "SB-4205: Voice of the customer"
summary: "Product hears about customers through anecdotes, not the tickets and notes already in the brain."
---

# SB-4205: Voice of the customer

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.2 |
| Stories | US-42.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-01, LAW-07, LAW-14 |
| Depends on | SB-4005 |
| Status | Open |

## Why

Product hears about customers through anecdotes, not the tickets and notes already in the brain.

## Acceptance criteria

- [ ] The `voc` agent can be a Product or Support space's weekly brief
- [ ] At most 8 themes, each with a count, a 4-week trend and three cites, from what the running steward can open
- [ ] `personal_data` fields stay masked (SB-1906)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent and prompt
- [ ] Catalogue
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
