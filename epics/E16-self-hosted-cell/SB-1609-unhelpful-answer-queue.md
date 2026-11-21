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
| Status | Open |

## Why

Thumbs-down feedback is stored but nobody sees it.

## Acceptance criteria

- [ ] `GET /v1/admin/feedback?rating=down` lists unhelpful answers with their trace's decisions and counts, never the question or the answer text
- [ ] The console shows the queue with the user's own comment; operators mark an item reviewed
- [ ] UI and API tests

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Endpoint
- [ ] Console queue
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
