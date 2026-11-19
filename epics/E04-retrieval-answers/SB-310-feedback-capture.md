---
title: "SB-310: Feedback capture"
summary: "The pilot is only useful if feedback is tied to a trace."
---

# SB-310: Feedback capture

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.3 Pilot |
| Stories | US-04.5 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | — |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md) |
| Status | In review |

## Why

The pilot is only useful if feedback is tied to a trace.

## Acceptance criteria

- [x] Thumbs and 'wrong citation' reports stored against the trace id
- [ ] Weekly export for review (the export and its window are built; scheduling it comes with the cell runtime)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Feedback endpoint
- [x] UI controls
- [x] Export job
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.7): `ask.Feedbacks.Give` stores a rating, an optional wrong citation and an optional comment against the trace. Only the asker may give feedback (auditors can read traces but not rate them). The wrong citation must be in that answer, and a later feedback replaces an earlier one. `POST /v1/feedback` answers `{"trace", "stored": true}` in the sb-api(3) shape. `ask.Week` and `Feedbacks.Export` write the previous Monday-to-Monday window as JSON lines of ids, counts and the comment, and a test checks that the question and answer text never appear. The web UI has Helpful and Not helpful buttons (`aria-pressed`), a wrong-citation picker limited to cited claims, and a comment field, all axe-clean. The memory trace store keeps feedback, keyed by tenant and trace.

**Left:** the weekly schedule and a Postgres store, both with the cell runtime.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
