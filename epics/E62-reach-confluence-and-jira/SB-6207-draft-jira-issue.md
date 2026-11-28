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
| Status | Done |

## Why

The next action can draft a Linear ticket but not a Jira issue.

## Acceptance criteria

- [x] The next action can draft a Jira issue: project, type, summary and a description with cites
- [x] The person files it with their own Atlassian token; the brain never files it
- [x] A draft citing what the project's readers can't open warns before it is filed

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Jira draft in `Agent.DraftAction`
- [x] Filing with the person's token
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `kind: jira-issue` on `POST /v1/actions/draft` returns the summary, text and a link to Jira's own create form, prefilled in the first project, so the person files it under their own sign-in, and a warning when it cites confidential or restricted documents; the web UI's **Draft a Jira issue**. The brain never files an issue and holds no token of the person's. Tested in TestAtlassianThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
