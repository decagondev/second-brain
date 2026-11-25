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
| Status | Done |

## Why

Product hears about customers through anecdotes, not the tickets and notes already in the brain.

## Acceptance criteria

- [x] The `voc` agent can be a Product or Support space's weekly brief
- [x] At most 8 themes, each with a count, a 4-week trend and three cites, from what the running steward can open
- [x] `personal_data` fields stay masked (SB-1906)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] Catalogue
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. The voc agent and prompt, a product or support space's brief (manifest spaceBrief); Agent.vocThemes keeps at most 8 themes with 3 cites each and counts this week and the 4 weeks before from the cited documents' first-seen dates. personal_data stays masked at ingest (SB-1906). Tests: TestFollowUpThroughARunningCell, agent-voc suite.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
