---
title: "SB-907: Identity Platform adapter for companies without an OIDC IdP"
summary: "Some acquired companies have no corporate IdP; they still need SSO, groups and fast deprovisioning."
---

# SB-907: Identity Platform adapter for companies without an OIDC IdP

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.3 Edge and identity |
| Stories | US-09.3 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-12](../../manual/sb-laws.7.md#law-12) |
| Depends on | [SB-106](../E01-tenancy-identity/SB-106-oidc-sign-in-per-tenant.md) |
| Status | In review |

## Why

Some acquired companies have no corporate IdP; they still need SSO, groups and fast deprovisioning.

## Acceptance criteria

- [ ] One Identity Platform tenant per company, federating that company's SAML or OIDC provider, behind the existing `oidc` port (the verifier is built; creating the Identity Platform tenants in a real project is SB-903)
- [x] Groups still come by SCIM; a company without SCIM gets a steward-managed group list in the manifest, never an open sign-in
- [x] Tokens from one company's Identity Platform tenant are refused by every other cell

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Adapter
- [x] Manifest field `idp: identity-platform://…`
- [x] Cross-tenant token test
- [x] Write or extend the law tests: LAW-02, LAW-12
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S9.3b): sb-identity(7) gained "Identity Platform, exactly", and sb-tenant(5) the `identity-platform://PROJECT/IDP_TENANT` idp and the `groups` field. The manifest parser accepts both; `groups:` is refused with an OIDC idp, and its group names, emails, duplicates and limits (200 groups, 2,000 members) are checked, and `Format` writes it back. `oidc.Provider.IdentityPlatform` sets the audience to the project and the expected Identity Platform tenant: a token whose `firebase.tenant` is missing or another company's is refused, and so is one without a verified email. The directory user is found by that email, never by Identity Platform's own user id. `internal/core/groupsync` applies `groups:` to the directory: listed people are active, people no longer listed are deactivated and their sessions revoked, and dropped groups are deleted. Two new law tests: `TestLAW02_IdentityPlatformTenantsStayApart` (one shared project, same issuer, keys and audience; a mutation that trusts the whole project is caught) and `TestLAW12_ManifestGroupRemovalRevokesFast` (removal refused within 5 minutes even with a lost revocation; a mutation that never removes is caught).

**Left:** creating each company's Identity Platform tenant in a real project, and calling `groupsync` from the in-cell apply (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
