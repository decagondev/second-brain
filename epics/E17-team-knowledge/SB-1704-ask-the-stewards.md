---
title: "SB-1704: Ask the stewards when nothing answers"
summary: "When nothing answers, the asker is told so and left there, and the person who knows never hears the question."
---

# SB-1704: Ask the stewards when nothing answers

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.2 |
| Stories | US-17.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-01, LAW-04, LAW-07, LAW-21 |
| Depends on | [SB-1701](SB-1701-search-without-the-model.md) |
| Status | Done |

## Why

When nothing answers, the asker is told so and left there, and the person who knows never hears the question.

## Acceptance criteria

- [x] On `no_sources`, `no_access` or `most_uncited` the UI offers to send the question to the space's stewards, or for a company-wide ask to the manifest's `questions_to` group; the asker sees exactly what is sent
- [x] Stewards see a queue on their screen and reply with text and cites; every cite is checked live for the asker, and one they can't open is dropped and counted
- [x] The asker sees replies in a Questions tab; unanswered questions age out at the space's retention; send and reply each write an audit row first
- [x] The weekly report counts questions asked and answered per space, never their text

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `questions_to` in the manifest
- [x] Queue, reply and the asker's list
- [x] Tests: a steward's cite can't widen access; a moved steward loses the queue
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.2): `ask.Stewards` with `POST/GET /v1/questions`, `GET /v1/spaces/NAME/questions` and `POST …/{id}/reply`, and the manifest field `questions_to` for company-wide questions. A question is sent only from the asker's own answer in the same space with `no_sources`, `no_access` or `most_uncited`; only its text goes, never the asker's name or the answer. Who works a queue is the new Cedar action `answer_questions` on the space, decided with the caller's groups at each request. A steward's cites must be ones they can open now; the asker sees each cite only after the same checks a search runs (`retrieve.Allowed`, split out of retrieval), the rest counted as `dropped`. Send and reply each write an `access` audit row with a hash first. Questions are records under retention (a space's shorter one included), held by holds and erased with the asker; the weekly report counts asked and answered per space. UI: "Ask the stewards of …" shows exactly what is sent; a Questions tab; the steward screen's queue. Tests: `TestLAW01_StewardRepliesShowOnlyWhatTheAskerCanOpen`, `TestLAW21_AMovedStewardLosesTheQueue`, `TestLAW04_QuestionsAreAuditedFirst` (self-tests catch an asker shown every cite and a rule that ignores groups), core and UI tests, and a running cell (`TestAnswerOnceThroughARunningCell`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
