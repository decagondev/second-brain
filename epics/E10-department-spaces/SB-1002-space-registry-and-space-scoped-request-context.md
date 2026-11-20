---
title: "SB-1002: Space registry and space-scoped request context"
summary: "Every request must know which space it's in, the way it already knows its tenant."
---

# SB-1002: Space registry and space-scoped request context

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.1 Space model |
| Stories | US-10.1 |
| Manual | `sb-spaces(7)` (new), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), LAW-19 (new, W8) |
| Depends on | [SB-1001](SB-1001-spaces-in-the-manifest-and-sb-tenant-check.md), [SB-108](../E01-tenancy-identity/SB-108-tenant-scoped-request-context.md) |
| Status | Done |

## Why

Every request must know which space it's in, the way it already knows its tenant.

## Acceptance criteria

- [x] Spaces stored in the cell DB with id `spc_<name>`; created and updated only by `sb tenant apply` (`spaces.Sync`; apply calls it once SB-1006 lifts the refusal)
- [x] The request context carries the space or `company`, set only by the auth middleware
- [x] API calls take `space`; an unknown space is `ESPACE`, never a fallback to company-wide

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Registry
- [x] Context
- [x] API parameter
- [x] Tests
- [x] Write or extend the law tests: LAW-02, LAW-19
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.1b): sb-spaces(7) gained "The space registry and the request context", sb-api(3) "Spaces", and sb-tenancy(7) the scope's space. `ports.SpaceStore` has a memory adapter and a Postgres one (`internal/adapters/postgres/spaces`, table `spaces` keyed by tenant and `spc_<name>`), both passing `spacestest.Contract` (Postgres against the WSL database). `spaces.Sync` makes the registry match a manifest, deleting dropped spaces; `sb tenant apply` is its only caller once SB-1006 lifts apply's refusal. `spaces.Resolve` turns `?space=` into a space id: none or `company` is company-wide, and an unknown space, a non-member (with the same message), another tenant's space and a missing registry are `ESPACE`; a registry that can't answer is `EUNAVAIL`. Two `space` values are `ESPACE`. `tenancy.Scope.Space()` carries it, set only through `Issuer.IssueIn` by the auth middleware. The new law test `TestLAW19_TheSpaceContextNeverWidens` drives the real middleware over a registry shared with another tenant and a registry that fails; a cell that falls back to company-wide is caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
