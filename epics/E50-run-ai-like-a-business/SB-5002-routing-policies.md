---
title: "SB-5002: Routing policies"
summary: "Every space pays the same for every task, whether it needs the largest model or not."
---

# SB-5002: Routing policies

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.1 |
| Stories | US-50.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06, LAW-07 |
| Depends on | SB-4405, SB-3902 |
| Status | Open |

## Why

Every space pays the same for every task, whether it needs the largest model or not.

## Acceptance criteria

- [ ] `routing:` (economy, balanced or frontier) per space, `balanced` by default, flows down the tree; each policy picks the documented tier per task
- [ ] The gate runs each space's eval suite on its policy and refuses a policy that fails
- [ ] A child can't choose a higher tier than its parent allows

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Setting and inheritance
- [ ] Router
- [ ] Gate check
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
