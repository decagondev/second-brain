---
title: "SB-602: Admin console where every action is a manifest diff"
summary: "One source of truth, two editors."
---

# SB-602: Admin console where every action is a manifest diff

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.1 Verify and console |
| Stories | US-07.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-104](../E01-tenancy-identity/SB-104-sb-tenant-apply-idempotent-cell-creation.md), [SB-403](../E05-actions-approvals-audit/SB-403-kill-switches-per-agent-connector-and-tenant.md) |
| Status | Open |

## Why

One source of truth, two editors.

## Acceptance criteria

- [ ] Connectors, agents, approval rules and kill switches editable
- [ ] Each change produces a manifest diff and an audit row
- [ ] Kill switches act immediately and offer 'make permanent' as a manifest diff

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Console UI
- [ ] Admin API
- [ ] Manifest writer
- [ ] Audit hooks
- [ ] Write or extend the law tests: LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
