---
title: "SB-1709: Documents to review for stewards"
summary: "The most damaging answer is a confident one from an old document, and nobody sees which old documents answer the most questions."
---

# SB-1709: Documents to review for stewards

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.4 |
| Stories | US-17.4 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14, LAW-16 |
| Depends on | — |
| Status | Open |

## Why

The most damaging answer is a confident one from an old document, and nobody sees which old documents answer the most questions.

## Acceptance criteria

- [ ] `review_after_days` in sb-space(5), default 365
- [ ] The steward's screen lists documents held by the space cited at least five times in 30 days and older than the threshold, plus any marked as a wrong citation, from the traces
- [ ] Each item is checked live as the steward; one they can't open is counted, with its owner if any, never named
- [ ] Cites gain `age_days`, and the UI shows a document's age on its chip past the threshold

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Counts from traces
- [ ] Steward section and chip age
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Planned 2026-11-22 with W16, accepted by the user the same day ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
