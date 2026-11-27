---
title: "SB-5903: Entra groups and people"
summary: "Access checks know Google groups but not Entra ones."
---

# SB-5903: Entra groups and people

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.1 |
| Stories | US-59.1 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-12 |
| Depends on | SB-107, SB-5404 |
| Status | Done |

## Why

Access checks know Google groups but not Entra ones.

## Acceptance criteria

- [x] A directory adapter for Entra ID: groups with transitive membership for access checks and audiences, managers for the tree and handovers
- [x] Entra SCIM provisioning into the existing endpoint revokes a leaver within 5 minutes
- [x] A group too large to expand is counted, never guessed

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `directory/entra` adapter
- [x] SCIM from Entra
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `msgraph.Directory`: `GroupsOf` (`transitiveMemberOf`, as `group:MAIL` and `group:ID`), `Members` (`transitiveMembers`, at most 5,000, `ErrTooLarge` past it), `Manager` and `Email` (an Entra object id's address, for Teams). Entra's SCIM provisioning goes into the existing endpoint, which takes its `Replace` and string booleans, so a leaver is signed out at once (LAW-12). Tested in TestEntraGroupsAndManagers, TestSharePointLiveCheck and TestMicrosoft365ThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
