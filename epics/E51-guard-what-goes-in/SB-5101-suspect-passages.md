---
title: "SB-5101: Suspect passages"
summary: "The sanitiser treats documents as data at ingest, but a passage that reads like instructions to the model is never shown to anyone."
---

# SB-5101: Suspect passages

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.1 |
| Stories | US-51.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-09, LAW-11 |
| Depends on | SB-207, SB-4406 |
| Status | Done |

## Why

The sanitiser treats documents as data at ingest, but a passage that reads like instructions to the model is never shown to anyone.

## Acceptance criteria

- [x] A retrieved passage addressed to the model is skipped, never quoted, and the answer says how many were skipped
- [x] Stewards see suspect passages on the review list (`section=suspect`) and the owner gets one notice per document
- [x] `suspect: monitor` logs the same events without skipping; the default is `skip`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Passage check at read time
- [x] Review list section and owner notice
- [x] Monitor setting
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `ingest.Suspect` says a passage reads like instructions: flagged `instruction_like` at ingest, or matching the instruction rules now (`ingest.InstructionLike`), so a chunk indexed before a rule was added is caught; stripped hidden text alone isn't suspect. `ask.Agent.screenIn` runs on every allowed context before the table and the model, and `composeWith` screens again for drafts and handover briefs, so no path gives a suspect passage to a model. A per-answer collector (`withSkips`, shared with consultations) keeps each suspect cite once: `Response.Skipped` counts the skipped ones and `Trace.Suspect` keeps the cites, never the text. A context left empty calls no model. sb-space(5) `suspect: skip|monitor` (`manifest.SuspectOf`, inherited from the department); `monitor` lets passages through wrapped and flagged and still records them. `Stewards.Suspect` serves `GET /v1/spaces/NAME/review?section=suspect` (documents with `passages`, checked as the steward, no snippet), and the notify job tells each owner once per document version (`notify.KindSuspect`, `Job.Suspects`, `Cell.suspects`). Tested in TestSuspectPassagesAreSkipped, TestSuspectMonitorRecordsOnly, TestOnlySuspectPassagesCallNoModel, TestSpaceSuspect, TestSuspectsTellOwnersOnce and TestGuardWhatGoesInThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
