---
title: "SB-4905: When a source changes"
summary: "A note or brief an agent drafted goes stale the day its source changes."
---

# SB-4905: When a source changes

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.2 |
| Stories | US-49.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-14, LAW-26 |
| Depends on | SB-4006, SB-4103 |
| Status | Open |

## Why

A note or brief an agent drafted goes stale the day its source changes.

## Acceptance criteria

- [ ] `on: source-changed` redrafts what an agent drafted from a changed document or policy in the last 90 days, with the change highlighted
- [ ] Changes are batched to at most one redraft a day, and nothing republishes without the approver
- [ ] A source the approver can no longer open drops the redraft for them

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Trigger
- [ ] Dependency lookup from traces
- [ ] Batching
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
