---
title: "SB-4206: Draft the fix"
summary: "A stale or conflicting document is flagged, but its owner still has to find what changed and write the fix."
---

# SB-4206: Draft the fix

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.2 |
| Stories | US-42.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-14, LAW-16 |
| Depends on | SB-3605 |
| Status | Done |

## Why

A stale or conflicting document is flagged, but its owner still has to find what changed and write the fix.

## Acceptance criteria

- [x] **Draft a fix** on a stale or conflicting document proposes a diff, each change citing its newer source
- [x] The suggestion goes to the owner as a message and to their inbox; the brain never writes the source
- [x] A source the owner can't open is not used; the suggestion is marked done when the source changes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] API
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. Legwork.DraftFix runs the fix-draft agent as the document's one active owner, keeps only changes citing another document, stores the suggestion for the owner (kind fix, suggested until the source has a newer version, then applied; LAW-26 on read) and tells them (fix, Legwork.Tell). POST /v1/spaces/NAME/review/fix; web Draft a fix on the review list. Nothing is written to the source. Tests: TestFollowUpThroughARunningCell, agent-fix-draft suite.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
