---
title: "SB-5201: Suggested edits to the source"
summary: "The brain finds a source stale or wrong, but can only tell the owner, not hand them a fix."
---

# SB-5201: Suggested edits to the source

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.1 |
| Stories | US-52.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-14, LAW-16 |
| Depends on | SB-2701, SB-3605, SB-4505, SB-3606 |
| Status | Done |

## Why

The brain finds a source stale or wrong, but can only tell the owner, not hand them a fix.

## Acceptance criteria

- [x] A stale, contradicted, replaced or reported source gets a drafted suggested edit for its owner: the passage, the change and the supporting cites
- [x] Nothing is written through a connector; the owner applies it in the source tool or dismisses it with a reason
- [x] A dismissed suggestion isn't made again for the same passage

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suggestion draft
- [x] Owner inbox and dismissal
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `Stewards.SuggestEdit` at `POST /v1/spaces/NAME/review/suggest` `{cite, why, evidence}`: the steward must open the document; its owner at the source must be one active user; the change is asked as the owner (`ScopeOf`), narrowed to the document and up to 4 documents of evidence, keeping no conversation, so it cites only what the owner can open. It lands in the owner's own queue as a `suggested-edit` draft with the passage's cite in `doc`, audited `suggested`; the steward hears only `{id, to: owner, doc}`. Nothing is written to a source. `POST /v1/me/suggestions/ID/dismiss` `{reason}` keeps the passage's cite on the dismissed draft, so neither an open nor a dismissed passage is suggested again. Started by a steward from the review list rather than by every check, so a person chooses which findings reach an owner. Tested in TestFixTheSourceThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
