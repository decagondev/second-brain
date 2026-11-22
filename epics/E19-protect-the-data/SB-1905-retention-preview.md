---
title: "SB-1905: Retention preview"
summary: "The nightly retention job deletes on schedule, and nobody can see what it will delete before it does."
---

# SB-1905: Retention preview

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.2 |
| Stories | US-19.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | LAW-16 |
| Depends on | — |
| Status | Open |

## Why

The nightly retention job deletes on schedule, and nobody can see what it will delete before it does.

## Acceptance criteria

- [ ] `GET /v1/admin/retention/plan?days=N` lists what the retention job would delete within N days, counted by kind, with records kept by a legal hold counted apart
- [ ] The plan runs the job's own selection with no delete, so the counts match what it then deletes
- [ ] `sb tenant retention --plan` prints it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Dry run in core/retention
- [ ] Endpoint and CLI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
