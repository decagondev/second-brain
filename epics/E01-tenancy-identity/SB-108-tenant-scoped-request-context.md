---
title: "SB-108: Tenant-scoped request context"
summary: "Tenant id must be impossible to forget, so it is a type, not a parameter."
---

# SB-108: Tenant-scoped request context

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.3 Identity |
| Stories | US-01.2 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-106](../E01-tenancy-identity/SB-106-oidc-sign-in-per-tenant.md) |
| Status | In review |

## Why

Tenant id must be impossible to forget, so it is a type, not a parameter.

## Acceptance criteria

- [x] Every repository and port method takes a `tenancy.Scope`; there is no unscoped constructor
- [x] LAW-02 green with two seeded tenants and adversarial queries (request-context level; retrieval level is SB-301)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Define `tenancy.Scope` with unexported fields
- [x] Thread it through ports
- [x] Adversarial cross-tenant query suite
- [ ] Turn LAW-02 green: the scope, sign-in, routing and cell tests are green; the retrieval-level test stays red until SB-301, as planned
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built.
- `internal/core/tenancy`: `Scope` has unexported fields. `Issuer.Issue` is the only constructor and refuses another tenant's session. The zero scope is invalid, and every scoped method checks it first. `WithScope`/`FromContext` carry it through a request.
- `auth.Cell.Middleware` authenticates the request, reads the user's groups from the directory, issues the scope and puts it in the context.
- CI (`make deps`) fails if `tenancy.NewIssuer` is called outside `internal/auth` or tests. The check itself had a bug, BUG-0005, found by mutation-checking it; it is fixed and has a regression test.
- The first cell-data port, `ports.People`, takes a `Scope` on every method. Ports may now import `core/tenancy` and nothing else (architecture table, depguard and `deps.py`).
- Sessions, the directory's auth side and SCIM run before a scope exists, by design. The registry and placement are control-plane ports with no tenant data. Every cell-data port from here on takes a `Scope` (SB-2xx, SB-3xx).

LAW-02 gains `TestLAW02_ScopeIsTheOnlyWayIn`, red first, then green. Other tenants' sessions get no scope, adversarial lookups of other tenants' ids find nothing, and a missing scope is an error. A directory that ignores the scope is caught.

**Left:** review by the user, (first CI run green on `c0a12be` (all jobs: check, proofs, postgres, diagrams)).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
