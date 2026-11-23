---
title: "SB-2205: Ask for access"
summary: "'N sources you can't open' tells the asker to ask the owner, but not who, and gives them no way to."
---

# SB-2205: Ask for access

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.3 |
| Stories | US-22.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-2101 |
| Status | Done |

## Why

"N sources you can't open" tells the asker to ask the owner, but not who, and gives them no way to.

## Acceptance criteria

- [x] Where an answer counts hidden sources, a button sends each of their owners one request; the owner's message names the document, the asker's view is a count
- [x] The asker never learns which document or who, unless already named in `people`
- [x] One request per asker, document and week

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Request endpoint
- [x] Owner message
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: built in S21.3. `Agent.RequestAccess` over the trace's `source_denied` decisions, the `access` notification by the chat bot, a hashed weekly ledger. Law tests: `TestLAW01_AskingForAccessTellsTheAskerOnlyCounts`, `TestLAW07_AnAccessRequestNeverCarriesTheQuestion`, self-test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
