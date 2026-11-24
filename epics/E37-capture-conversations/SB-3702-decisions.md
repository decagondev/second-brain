---
title: "SB-3702: Decisions"
summary: "Decisions are buried in documents with no way to list them."
---

# SB-3702: Decisions

| Field | Value |
|---|---|
| Epic | [E37 Capture Conversations](README.md) |
| Wave / slice | W36 / S36.1 |
| Stories | US-37.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01 |
| Depends on | SB-2404 |
| Status | Done |

## Why

Decisions are buried in documents with no way to list them.

## Acceptance criteria

- [x] A section headed Decision or Decisions, or front matter `type: decision`, is typed decision
- [x] `GET /v1/search?type=decision` lists them newest first; Documents gets a Decisions filter
- [x] Other documents are unchanged

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ingest
- [x] Search and UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Decision chunks typed decision; type=decision newest first, listed with no query in a space; Decisions filter

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
