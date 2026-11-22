---
title: "SB-1609: A queue of answers people marked unhelpful"
summary: "Thumbs-down feedback is stored but nobody sees it."
---

# SB-1609: A queue of answers people marked unhelpful

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.4 |
| Stories | US-16.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14 |
| Depends on | — |
| Status | Done |

## Why

Thumbs-down feedback is stored but nobody sees it.

## Acceptance criteria

- [x] `GET /v1/admin/feedback?rating=down` lists unhelpful answers with their trace's decisions and counts, never the question or the answer text
- [x] The console shows the queue with the user's own comment; operators mark an item reviewed
- [x] UI and API tests

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Endpoint
- [x] Console queue
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S15.4): `GET /v1/admin/feedback?rating=down` lists the last 90 days of unhelpful answers, newest first: ids, counts, the trace's retrieval decisions counted by verdict and reason, and the asker's own comment; never the question, the answer or a cite, so nothing of a document the operator couldn't open. `POST /v1/admin/feedback/TRACE/reviewed` writes a `config` audit row first and then marks the item reviewed; a second review keeps the first, and a later feedback puts the item back. Both stores gained `GetFeedback`. The console shows the queue after health, with a button to mark each reviewed. Tests: the core queue (`TestQueueAndReview`), a running cell (`TestUnhelpfulAnswerQueue`, which checks the queue holds none of the question's or the answer's words), and the console with axe.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
