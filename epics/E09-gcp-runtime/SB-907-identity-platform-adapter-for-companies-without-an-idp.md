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
| Status | Open |

## Why

Some acquired companies have no corporate IdP; they still need SSO, groups and fast deprovisioning.

## Acceptance criteria

- [ ] One Identity Platform tenant per company, federating that company's SAML or OIDC provider, behind the existing `oidc` port
- [ ] Groups still come by SCIM; a company without SCIM gets a steward-managed group list in the manifest, never an open sign-in
- [ ] Tokens from one company's Identity Platform tenant are refused by every other cell

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Adapter
- [ ] Manifest field `idp: identity-platform://…`
- [ ] Cross-tenant token test
- [ ] Write or extend the law tests: LAW-02, LAW-12
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
