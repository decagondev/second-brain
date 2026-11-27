---
title: "SB-6207: Draft a Jira issue"
summary: "The next action can draft a Linear ticket but not a Jira issue."
---

# SB-6207: Draft a Jira issue

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.2 |
| Stories | US-62.2 |
| Manual | [`sb-actions(3)`](../../manual/sb-actions.3.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-16, LAW-32 |
| Depends on | SB-5304, SB-6204 |
| Status | Open |

## Why

The next action can draft a Linear ticket but not a Jira issue.

## Acceptance criteria

- [ ] The next action can draft a Jira issue: project, type, summary and a description with cites
- [ ] The person files it with their own Atlassian token; the brain never files it
- [ ] A draft citing what the project's readers can't open warns before it is filed

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Jira draft in `Agent.DraftAction`
- [ ] Filing with the person's token
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
