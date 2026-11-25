---
title: "SB-5207: `@` and `/` in the ask box"
summary: "Asking about a space, person or document, or using a skill, means knowing its exact name."
---

# SB-5207: `@` and `/` in the ask box

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.2 |
| Stories | US-52.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01 |
| Depends on | SB-3004, SB-4601, SB-4906 |
| Status | Done |

## Why

Asking about a space, person or document, or using a skill, means knowing its exact name.

## Acceptance criteria

- [x] `@` offers spaces, people and documents the person can open and adds them to the ask
- [x] `/` offers the space's skills and starter questions
- [x] Every suggestion is checked as the person; a document they can't open is never offered

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suggest endpoint
- [x] Ask box
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built in the web app, from endpoints that already check as the asker: `@WORD` offers the asker's spaces (`GET /v1/spaces`), the space's stewards and owners (`GET /v1/spaces/NAME/people`) and up to 5 documents (`GET /v1/search`); picking one asks in the space, lists the person's documents, or narrows the ask to the document, and takes the `@WORD` out. `/` offers the space's starter questions beside its skills. No new endpoint. Tested in web `athint.test.tsx` and `shape.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
