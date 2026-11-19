---
title: "SB-107: SCIM provisioning and fast deprovisioning"
summary: "A leaver must lose access within minutes, not at the next sync."
---

# SB-107: SCIM provisioning and fast deprovisioning

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.3 Identity |
| Stories | US-01.3 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | [LAW-12](../../manual/sb-laws.7.md#law-12) |
| Depends on | [SB-106](../E01-tenancy-identity/SB-106-oidc-sign-in-per-tenant.md) |
| Status | Done |

## Why

A leaver must lose access within minutes, not at the next sync.

## Acceptance criteria

- [x] SCIM create, update, deactivate supported
- [x] Deactivate revokes sessions in 5 minutes or less (LAW-12 test)
- [x] Group membership changes reach policy within the same bound

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] SCIM endpoint per tenant
- [x] Session revocation fan-out
- [x] LAW-12 timed test
- [x] Update ledger
- [x] Write or extend the law tests: LAW-12
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built.
- `internal/scim` serves SCIM 2.0 at `/scim/v2` for the cell's tenant: Users (create, get, replace, patch, delete-as-deactivate) and Groups (create, replace, patch members, delete). It uses a bearer token compared in constant time, and accepts the `"False"` string some IdPs send.
- Going from active to inactive revokes the user's sessions in the response path. A failed revocation is a `503`, so the IdP retries.
- `ports.DirectoryStore` adds users, groups and `GroupsOf`. Policy reads groups at decision time, so membership changes apply to the next decision. `internal/adapters/directory/memory` is the per-cell store for now.

LAW-12 has its first law test: red first, then green. It runs on a fake clock, with and without a lost revocation fan-out, and also requires that other users keep working. A cell that ignores the directory after a lost fan-out is caught.

2026-11-19: reviewed by the user; CI green. **Done.**

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
