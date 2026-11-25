---
title: "SB-5202: Edit a draft together"
summary: "Reviewers can comment on a draft but not edit it, and the agent's changes arrive all at once."
---

# SB-5202: Edit a draft together

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.1 |
| Stories | US-52.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14, LAW-16, LAW-28 |
| Depends on | SB-4602, SB-4902, SB-4706 |
| Status | Done |

## Why

Reviewers can comment on a draft but not edit it, and the agent's changes arrive all at once.

## Acceptance criteria

- [x] Reviewers edit any agent draft in place, each edit kept with the draft's audit trail
- [x] **Ask for changes** returns suggestions accepted or rejected one by one; a rejected one leaves the text unchanged
- [x] A draft with open suggestions can't publish, and the note is marked `edited` with its AI mark kept

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Edits on drafts
- [x] Suggestions
- [x] Publish rule
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `SavedThread.Suggestions` (`DraftSuggestion`: section, old, new, round, state), `Generated` (the agent's text before an in-place edit) and `Editors`. Ask for changes still writes its text into the draft (so earlier rounds' versions and diffs are unchanged), and `suggest` records each redrafted section, or the whole draft, as an open suggestion; a later round accepts the open ones it built on. `Stewards.Decide` at `POST …/suggestions/N/accept|reject` (reject puts back what the section held; a section edited since can't be rejected), `Stewards.EditSection` at `PUT …/sections/N`. `Note` refuses a draft with an open suggestion, and `markFront` marks `edited` against `Generated` when the reviewer edited in place. Web: `Suggestions` beside Ask for changes. Tested in TestEditADraftTogether, TestFixTheSourceThroughARunningCell and web `shape.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
