---
title: "SB-4604: Post-incident review"
summary: "After an incident, someone rebuilds the timeline by hand and follow-ups get lost."
---

# SB-4604: Post-incident review

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.2 |
| Stories | US-46.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-01, LAW-09, LAW-16 |
| Depends on | SB-4204, SB-4203 |
| Status | Done |

## Why

After an incident, someone rebuilds the timeline by hand and follow-ups get lost.

## Acceptance criteria

- [x] A closed ticket that had an incident brief drafts one review to the steward queue
- [x] The review holds a timeline, decisions, changes and follow-ups as action items the steward confirms
- [x] An instruction in the ticket is ignored; nothing is posted to the ticket

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] Trigger
- [x] Eval suite
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. The Linear connector writes a `Status: NAME` line after `Labels:`, with ` (closed)` when the workflow state is completed or canceled (`lineartest` gained `Tag`). `ask.IsClosed` reads it among the first five lines. The notify job's incident pass (`notify.incidents`) now runs `RunReview` for a new version of a ticket whose brief ran in an earlier run and that is closed, once per ticket and space (clock key `review-…`), and tells the space's stewards "An incident review is waiting in SPACE". `wire.runReview` runs `incident-review` as the space's first active steward through `runDraft`, which is `runAsIn` for one document: the draft (`DraftReview`, "Review: TITLE") carries the ticket as its `doc` and the ticket's `Action:` lines as action items, read without a model by `DraftActions`; `ConfirmActions` accepts a review as it does a decisions draft. Agent definition, prompt `incident-review.md`, suite `agent-incident-review` (with an injection case and a fixture ticket that carries one), gate entry and the base policy's legwork list. Nothing is posted to the ticket. Tests: `tests/cell/further_test.go` (no review while open, one when closed, its action confirmed, no second), `linear_test.go` (`TestLabelsAndStatusLead`), `ask/further_internal_test.go` (`TestIsClosed`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
