---
title: "SB-1205: SCIM attributes and five-minute moves"
summary: "When someone changes team, their access should change with them, quickly."
---

# SB-1205: SCIM attributes and five-minute moves

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.3 Reorgs |
| Stories | US-12.3 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | [LAW-12](../../manual/sb-laws.7.md#law-12), LAW-21 (new, W8) |
| Depends on | [SB-107](../E01-tenancy-identity/SB-107-scim-provisioning-and-fast-deprovisioning.md) |
| Status | Done |

## Why

When someone changes team, their access should change with them, quickly.

## Acceptance criteria

- [x] SCIM enterprise attributes (`department`, `division`, `costCenter`, `manager`) and title level are stored and passed to Cedar
- [x] Leaving a space's group revokes that space within 5 minutes on a fake clock; joining grants only after the cell confirms the membership
- [x] LAW-21 green

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Attribute mapping
- [x] Move handling
- [x] LAW-21 law test green
- [x] Write or extend the law tests: LAW-12, LAW-21
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.3a): sb-identity(7) gained "Attributes and moves, exactly", sb-permissions(7) "Who the asker is, now", and sb-policy(5) the `User` attributes. SCIM keeps the core `title` and the enterprise `department`, `division`, `costCenter` and `manager` (POST, PUT and PATCH by full path); `ports.User.Attributes` maps them, `auth.Cell.Attributes` (`auth.DirectoryAttributes`) reads them at every request, `Scope.WithAttrs` carries them, and the kernel passes them to Cedar as the principal's optional attributes (schema updated), so a rule can use them with `has`. A directory that can't give them refuses with `EUNAVAIL`. Moves needed no cache to expire: the middleware reads groups from the directory at each request, and nothing about the asker comes from the token. `TestLAW21_MovesRevokeFast` now runs through the real SCIM endpoint and the auth middleware on a fake clock and is green, out of the ratchet; groups cached across requests, and groups taken from the IdP's claims, are caught. "Title level" is the SCIM `title`: SCIM has no standard level, so none is invented.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
