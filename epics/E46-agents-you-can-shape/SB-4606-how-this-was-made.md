---
title: "SB-4606: How this was made"
summary: "Nobody can see which agent, skill, model or departments stood behind an answer."
---

# SB-4606: How this was made

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.2 |
| Stories | US-46.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-01, LAW-11, LAW-26 |
| Depends on | SB-4501, SB-4601, SB-4405 |
| Status | Open |

## Why

Nobody can see which agent, skill, model or departments stood behind an answer.

## Acceptance criteria

- [ ] Every answer and draft has **How this was made**: agent and version, skill, tier, departments consulted, sub-questions, sources, dropped cites and cost
- [ ] Sources the viewer can't open are counted, not named
- [ ] Every field matches the trace and is checked as the viewer on opening

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
