---
title: "SB-4102: Status updates"
summary: "Weekly updates are written by hand from what the brain already knows."
---

# SB-4102: Status updates

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.1 |
| Stories | US-41.1 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-16 |
| Depends on | SB-4005 |
| Status | Done |

## Why

Weekly updates are written by hand from what the brain already knows.

## Acceptance criteria

- [x] A space with `updates: true` gets a draft update each Monday from the `update` agent: what changed, new decisions and goals' status, each line cited
- [x] The steward edits and publishes it as a note under `notes/SPACE/updates/`
- [x] The parent's Start tab lists its children's latest published updates, each checked as the reader

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent prompt
- [x] Notify job
- [x] Notes
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. Space updates: true; the update agent (prompts/set/update.md, base.cedar use_agent) drafts every Monday through notify Updates/RunUpdate as the first active steward; published to notes/NAME/updates/; GET /v1/spaces/NAME/updates lists each team's latest, checked as the caller; agent-update eval suite in the gate; web TeamUpdates.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
