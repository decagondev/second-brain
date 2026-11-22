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
| Status | Open |

## Why

When nothing answers, the asker is told so and left there, and the person who knows never hears the question.

## Acceptance criteria

- [ ] On `no_sources`, `no_access` or `most_uncited` the UI offers to send the question to the space's stewards, or for a company-wide ask to the manifest's `questions_to` group; the asker sees exactly what is sent
- [ ] Stewards see a queue on their screen and reply with text and cites; every cite is checked live for the asker, and one they can't open is dropped and counted
- [ ] The asker sees replies in a Questions tab; unanswered questions age out at the space's retention; send and reply each write an audit row first
- [ ] The weekly report counts questions asked and answered per space, never their text

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `questions_to` in the manifest
- [ ] Queue, reply and the asker's list
- [ ] Tests: a steward's cite can't widen access; a moved steward loses the queue
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Planned 2026-11-22 with W16, accepted by the user the same day ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
