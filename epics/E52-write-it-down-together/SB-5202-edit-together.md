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
| Status | Open |

## Why

Reviewers can comment on a draft but not edit it, and the agent's changes arrive all at once.

## Acceptance criteria

- [ ] Reviewers edit any agent draft in place, each edit kept with the draft's audit trail
- [ ] **Ask for changes** returns suggestions accepted or rejected one by one; a rejected one leaves the text unchanged
- [ ] A draft with open suggestions can't publish, and the note is marked `edited` with its AI mark kept

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Edits on drafts
- [ ] Suggestions
- [ ] Publish rule
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
