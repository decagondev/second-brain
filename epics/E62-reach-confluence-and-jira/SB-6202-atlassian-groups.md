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
| Status | Done |

## Why

Confluence and Jira permissions name Atlassian groups the cell can't resolve.

## Acceptance criteria

- [x] A directory adapter resolves Atlassian groups and members behind every access check and audience
- [x] SCIM from Atlassian Guard into the existing endpoint revokes a leaver within 5 minutes
- [x] A group the API won't expand is counted, never guessed

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Atlassian directory adapter
- [x] SCIM mapping
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `atlassian.Directory`: an active account by email, a person's groups, a group's members in pages of 50, more than 5,000 `ErrTooLarge`. The live checks ask Confluence and Jira themselves, so leaving a group is refused at the next ask; a leaver is revoked through the IdP's SCIM as before (LAW-12). Atlassian Guard provisions into Atlassian from the IdP, not out of it, so there is no second SCIM feed to take. Tested in TestConfluence; `sb prove SB-6202` covers it.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
