---
title: "SB-4902: Comments on a section"
summary: "A reviewer can only comment on a whole draft, so a redraft can change what was already right."
---

# SB-4902: Comments on a section

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.1 |
| Stories | US-49.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | LAW-09, LAW-14, LAW-16 |
| Depends on | SB-4602 |
| Status | Done |

## Why

A reviewer can only comment on a whole draft, so a redraft can change what was already right.

## Acceptance criteria

- [x] Reviewers comment on a section of any agent draft; comments are visible to reviewers only and kept in the audit trail
- [x] **Ask for changes** redrafts only commented sections; an uncommented section is unchanged byte for byte
- [x] A redraft cites only what the first could, a comment is never a source, and rounds still stop at 3

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Sections
- [x] Comments
- [x] Section redraft and diff
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `internal/core/ask/sections.go`: `DraftSections` splits a draft at headings, list items and paragraphs so the parts join back exactly (TestDraftSections, FuzzDraftSections). `POST /v1/spaces/NAME/questions/{id}/comments` adds the reviewer's comment on a section (the steward the draft ran as, the only one who sees its text), audited by hash and kept on the draft with its round; the queue shows `sections` and this round's `comments` to them only. Ask for changes with comments writes each commented section alone, from the first draft's sources, with its comments in the reviewer-note block, and keeps every other section byte for byte; the answer has a diff per section; rounds still stop at 3. Tested in TestShapeTheWorkThroughARunningCell. The web UI has no comment control yet; comments are made over the API.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
