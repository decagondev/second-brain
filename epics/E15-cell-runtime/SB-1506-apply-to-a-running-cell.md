---
title: "SB-1506: sb tenant apply against a running cell"
summary: "Apply plans and checks offline; a running cell must take the manifest and match it."
---

# SB-1506: `sb tenant apply` against a running cell

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.3 |
| Stories | US-15.1 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | LAW-16 |
| Depends on | [SB-1502](SB-1502-api-service.md) |
| Status | Open |

## Why

Apply plans and checks offline; a running cell must take the manifest and match it.

## Acceptance criteria

- [ ] Apply sends the checked manifest to the cell, which syncs spaces (refusing a held space), groups, workflows and duties, and audits the change
- [ ] Drift is reported hourly from the running cell (sb-tenant(5), Drift)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Apply endpoint and client
- [ ] Space and group sync
- [ ] Drift reporter
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
