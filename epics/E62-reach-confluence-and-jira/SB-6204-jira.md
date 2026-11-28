---
title: "SB-6204: Jira"
summary: "Teams that track work in Jira get nothing from it."
---

# SB-6204: Jira

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.2 |
| Stories | US-62.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-6202, SB-4204, SB-4604 |
| Status | In review |

## Why

Teams that track work in Jira get nothing from it.

## Acceptance criteria

- [x] A Jira Cloud connector reads issues, comments and attachments of the projects named, with links back
- [x] Permission schemes and issue security levels resolve to Atlassian groups and roles
- [x] Labels and status are read as Linear's are; a deleted issue is gone at the next sync

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Jira adapter and recorded fake
- [x] Security levels to ACLs
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `tickets: jira` (`atlassian.Jira`): projects' issues listed with their updated times, summary, description and comments as text with `Labels:` and `Status:` lines as Linear's, the people Jira says may browse each one as its ACL copy (`jira:users` past 1,000 without a security level), Jira's permission search for the person as the live check. Tested in TestJira and TestAtlassianThroughARunningCell; in review until a real site run.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
