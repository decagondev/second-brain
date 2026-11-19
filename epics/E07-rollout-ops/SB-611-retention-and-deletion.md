---
title: "SB-611: Retention and deletion"
summary: "Retention is configuration, not a project."
---

# SB-611: Retention and deletion

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.3 Operate |
| Stories | US-07.5 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-209](../E03-connectors-ingestion/SB-209-deletion-and-acl-change-propagation.md) |
| Status | Open |

## Why

Retention is configuration, not a project.

## Acceptance criteria

- [ ] `retention_days` enforced on derived data and logs; audit retention follows its own policy
- [ ] Subject deletion request removes derived data and records the deletion in audit

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Retention jobs
- [ ] Deletion API
- [ ] Tests
- [ ] Write or extend the law tests: LAW-13
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
