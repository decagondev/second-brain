---
title: "SB-6202: Atlassian groups and people"
summary: "Confluence and Jira permissions name Atlassian groups the cell can't resolve."
---

# SB-6202: Atlassian groups and people

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.1 |
| Stories | US-62.1 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-03, LAW-12 |
| Depends on | SB-107, SB-5404 |
| Status | Open |

## Why

Confluence and Jira permissions name Atlassian groups the cell can't resolve.

## Acceptance criteria

- [ ] A directory adapter resolves Atlassian groups and members behind every access check and audience
- [ ] SCIM from Atlassian Guard into the existing endpoint revokes a leaver within 5 minutes
- [ ] A group the API won't expand is counted, never guessed

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Atlassian directory adapter
- [ ] SCIM mapping
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
