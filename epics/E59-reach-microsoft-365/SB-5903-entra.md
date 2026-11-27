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
| Status | Open |

## Why

Access checks know Google groups but not Entra ones.

## Acceptance criteria

- [ ] A directory adapter for Entra ID: groups with transitive membership for access checks and audiences, managers for the tree and handovers
- [ ] Entra SCIM provisioning into the existing endpoint revokes a leaver within 5 minutes
- [ ] A group too large to expand is counted, never guessed

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `directory/entra` adapter
- [ ] SCIM from Entra
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
