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
| Status | Open |

## Why

Decisions are buried in documents with no way to list them.

## Acceptance criteria

- [ ] A section headed Decision or Decisions, or front matter `type: decision`, is typed decision
- [ ] `GET /v1/search?type=decision` lists them newest first; Documents gets a Decisions filter
- [ ] Other documents are unchanged

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ingest
- [ ] Search and UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
