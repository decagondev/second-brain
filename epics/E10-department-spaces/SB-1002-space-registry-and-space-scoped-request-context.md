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
| Status | Open |

## Why

Every request must know which space it's in, the way it already knows its tenant.

## Acceptance criteria

- [ ] Spaces stored in the cell DB with id `spc_<name>`; created and updated only by `sb tenant apply`
- [ ] The request context carries the space or `company`, set only by the auth middleware
- [ ] API calls take `space`; an unknown space is `ESPACE`, never a fallback to company-wide

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Registry
- [ ] Context
- [ ] API parameter
- [ ] Tests
- [ ] Write or extend the law tests: LAW-02, LAW-19
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
