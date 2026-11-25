---
title: "SB-4602: Ask for changes"
summary: "A draft that's nearly right can only be published as is, edited by hand or thrown away."
---

# SB-4602: Ask for changes

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.1 |
| Stories | US-46.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | LAW-09, LAW-14, LAW-16 |
| Depends on | SB-4005, SB-4006 |
| Status | Done |

## Why

A draft that's nearly right can only be published as is, edited by hand or thrown away.

## Acceptance criteria

- [x] **Ask for changes** with a note of at most 500 characters redrafts from the same sources and shows the diff
- [x] At most 3 rounds; a fourth is refused
- [x] The redraft cites nothing the first draft couldn't; the note is never a source

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Redraft
- [x] API
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. An agent draft keeps what it asked (`SavedThread.Asked`, `AskDocs`) and the documents it cited or offered (`Sources`, `draftSources`, at most 40). `POST /v1/spaces/NAME/questions/{id}/changes {note}` (`Stewards.AskForChanges`), for the steward the draft ran as only, redrafts through the same `draftAnswer` as the first draft, as them now (`Stewards.Rescope`, with their branch for a leader's week), narrowed to those sources (`withOnly`, the catch-up hook renamed) or to a decisions draft's transcript, so a redraft cites nothing the first couldn't. The note goes to the model with the earlier text in one `reviewer-note` data block (`noteData`), never a source or a cite. Every earlier text is kept (`Versions`), `Rounds` counts them, a fourth is `EINVAL`, and each round writes a `redrafted` audit row with the note's hash. The queue shows `versions`, `rounds` and the line `diff` (`ask.LineDiff`, moved from the CLI) to the steward it ran as only. Web: **Ask for changes** with the diff on each agent draft (`AskForChanges`). Tests: `tests/cell/shape_test.go` (another steward, a long note, three rounds whose every cite is a first source, a fourth refused), `web/src/shape.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
