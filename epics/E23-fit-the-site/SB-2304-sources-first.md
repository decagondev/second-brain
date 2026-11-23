---
title: "SB-2304: Search results while the answer is written"
summary: "On a CPU box answers take 7 to 32 seconds with nothing on screen."
---

# SB-2304: Search results while the answer is written

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.2 |
| Stories | US-23.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Done |

## Why

On a CPU box answers take 7 to 32 seconds with nothing on screen.

## Acceptance criteria

- [x] The web UI asks search and ask together and shows the allowed sources (title, owner, age) at once, then the cited answer
- [x] The sources are search's, checked as the asker, and are never shown as claims
- [x] With a slow fake model the sources render first; axe passes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: done in S22.2. Asking sends `GET /v1/search` with `POST /v1/ask`; while the answer is written the turn shows search's allowed sources by title, owner and age (`web/src/Sources.tsx`), never their text and never as claims, and the cited answer replaces them. A failed search changes nothing. `sources.test.tsx` holds the model back and checks the sources come first, no claim or snippet shows, and axe passes before and after. Manual: sb-ask(3).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
