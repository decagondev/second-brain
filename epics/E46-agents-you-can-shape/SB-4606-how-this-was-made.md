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
| Status | Done |

## Why

Nobody can see which agent, skill, model or departments stood behind an answer.

## Acceptance criteria

- [x] Every answer and draft has **How this was made**: agent and version, skill, tier, departments consulted, sub-questions, sources, dropped cites and cost
- [x] Sources the viewer can't open are counted, not named
- [x] Every field matches the trace and is checked as the viewer on opening

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. Every trace keeps `cost_usd`: `ask` opens its own `budget.Tally`, which now adds to the tally above it, so a report's cap still counts every call and a consultation's cost is inside its answer's. `GET /v1/traces/{id}/how` (`Agent.How`) and `GET /v1/research/{id}/how` (`Legwork.How`) work it out when opened, from the trace (and a report's section traces): agent, release, skill, tier, steps, departments, not consulted, refused, dropped and cost; sources are the trace's allowed documents, checked as the reader now by `docTitles`, with the rest counted in `sources_hidden` and never named. An agent draft's view carries its `trace` for the steward it ran as only. Web: **How this was made** (`web/src/Further.tsx`) under each answer, agent draft and report, fetched only when opened. Tests: `tests/cell/further_test.go` (a source the reader loses is counted, not named; someone else's report is refused), `budget/tally_test.go`, `web/src/further.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
