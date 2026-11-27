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
| Status | Open |

## Why

Teams that track work in Jira get nothing from it.

## Acceptance criteria

- [ ] A Jira Cloud connector reads issues, comments and attachments of the projects named, with links back
- [ ] Permission schemes and issue security levels resolve to Atlassian groups and roles
- [ ] Labels and status are read as Linear's are; a deleted issue is gone at the next sync

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Jira adapter and recorded fake
- [ ] Security levels to ACLs
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
