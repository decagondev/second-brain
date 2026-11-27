---
title: "SB-5603: Answered before, checked now"
summary: "The same question asked twice in a day costs twice and can come back different."
---

# SB-5603: Answered before, checked now

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.1 |
| Stories | US-56.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-4606, SB-4201 |
| Status | Done |

## Why

The same question asked twice in a day costs twice and can come back different.

## Acceptance criteria

- [x] An ask whose normalised question, space, scope and language match one answered in the last 7 days reuses it when every cite's document is unchanged and the asker can open all of them now
- [x] A reused answer says "answered on DATE, checked now", offers **Ask fresh** and costs no model call
- [x] Memory, exclusions, kept or dropped documents, a file or a collection make an ask unique and never reuse

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Reuse lookup from traces
- [x] Recheck as the asker
- [x] Web: **Ask fresh**
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ask.ReuseStore` (memory and Postgres) keeps a first, fully cited answer under `reuseKey` (normalised question, space, kinds, language, agent, length, verified). `Agent.reusedFrom` reuses it within 7 days, with no model call, when the documents this ask's own live retrieval allows (checked as the asker, after retirement, holds and exclusions) are exactly the ones it was composed from, the same versions (`Trace.Read`); its cites are dressed again for now; `mayReuse` rules out memory, exclusions, keep or drop, a file, a collection, documents, a skill, a consult or a choice. `Response.Reused` and **Ask fresh** (`fresh`). Tested in TestAnsweredBeforeCheckedNow and TestKeepAnsweringThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
