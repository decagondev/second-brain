---
title: "SB-3301: See also"
summary: "An answer shows only what it cites, so the near-misses the search found are lost."
---

# SB-3301: See also

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.1 |
| Stories | US-33.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | SB-3001 |
| Status | Done |

## Why

An answer shows only what it cites, so the near-misses the search found are lost.

## Acceptance criteria

- [x] The response's `see_also` lists at most 3 documents retrieval kept for the asker that no claim cites, by title, cite and age, never their text
- [x] The UI lists them as See also with Show passage and Ask about this document; chat leaves them out
- [x] A document the asker can't open never appears

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

ask.SeeAlso and Response.SeeAlso: at most 3 documents of the allowed context no claim cites, in rank order, each once, with span and age, never text; spanOf opens their passage; the UI's SeeAlsoView with Show passage and Ask about this document; chat leaves them out. Tests: ask/askbetter_test.go TestSeeAlso, web askbetter.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
