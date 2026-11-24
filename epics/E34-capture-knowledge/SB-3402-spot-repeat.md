---
title: "SB-3402: Spot a repeat"
summary: "Stewards answer the same question twice, because nothing spots a repeat."
---

# SB-3402: Spot a repeat

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.1 |
| Stories | US-34.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-1704 |
| Status | Done |

## Why

Stewards answer the same question twice, because nothing spots a repeat.

## Acceptance criteria

- [x] A new question to the stewards close to one of the space's last 90 days is shown to the stewards as Looks like q_N
- [x] Answer both sends one reply to each asker, each checked as its asker
- [x] No asker sees another's question

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Matching
- [x] Steward screen
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. At send, the cell embeds the question with its embedder and compares it with the newest 200 questions of the same queue from the last 90 days; the closest at cosine 0.85 or more is kept as `like`. The queue shows it both ways, to stewards only, as **Looks like qs_…**; **Answer both** is the reply with `both: true`, one reply and one audit row per question. No embedder, or one that fails, links nothing. Tests: `TestRepeatsAndAnswerBoth`, web `capture.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
