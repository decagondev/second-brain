---
title: "SB-4502: Hand off to a department"
summary: "Work that belongs to another department dead-ends in an answer, and the asker starts again there."
---

# SB-4502: Hand off to a department

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.1 |
| Stories | US-45.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-16 |
| Depends on | SB-4104, SB-4501 |
| Status | Done |

## Why

Work that belongs to another department dead-ends in an answer, and the asker starts again there.

## Acceptance criteria

- [x] An answer that matches another space's request category offers **Hand off**, drafting the category, text and the cites the asker ticks
- [x] Nothing reaches the department until the asker sends it
- [x] The receiving steward sees only cites they can open; the rest are counted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Draft
- [x] API
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. An answer whose question names, as whole words in any case, a request category of another space the asker is a member of carries `handoff` (`ask.Agent.Requests`, the first space by name). `POST /v1/spaces/NAME/handoff` (`ask.Stewards.HandoffDraft`) drafts the request from the asker's own trace: the category, the question as text, and the answer's cited documents the asker can open now; it stores and sends nothing. `POST /v1/spaces/NAME/requests` takes `cites` (`Stewards.RequestWith`, at most `ask.MaxShared` 10, each one the sender can open now, or EINVAL), kept as `StewardQuestion.Shared`. The steward queue shows `shared` (cites the steward can open now, with titles) and `shared_hidden` (the rest, counted), checked live on every read. Web: **Hand off** on the answer, a draft with the cites to tick, and Send; the queue lists shared sources. The cell test covers the draft, a cite the sender can't open, and a steward who can open none of them.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
