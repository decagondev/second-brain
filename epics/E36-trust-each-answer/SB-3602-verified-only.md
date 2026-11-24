---
title: "SB-3602: Verified only"
summary: "There is no way to ask only from documents someone has checked."
---

# SB-3602: Verified only

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.1 |
| Stories | US-36.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-2501 |
| Status | Done |

## Why

There is no way to ask only from documents someone has checked.

## Acceptance criteria

- [x] `POST /v1/ask` and `GET /v1/search` take `verified: true`, keeping only documents with a live mark at their current version, and notes
- [x] The UI's Verified only toggle is kept in the browser
- [x] Nothing verified matching says so, as found-nothing does

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retrieval
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Verified only on ask and search, from marked documents and notes alone

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
