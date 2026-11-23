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
| Status | Done |

## Why

The nightly retention job deletes on schedule, and nobody can see what it will delete before it does.

## Acceptance criteria

- [x] `GET /v1/admin/retention/plan?days=N` lists what the retention job would delete within N days, counted by kind, with records kept by a legal hold counted apart
- [x] The plan runs the job's own selection with no delete, so the counts match what it then deletes
- [x] `sb tenant retention --plan` prints it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Dry run in core/retention
- [x] Endpoint and CLI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.2): `retention.Job.Plan(in)` runs the sweep's own selection (`Store.Plan` beside each `Expire`, the tenant's cutoff and then each shorter space's, counting only the records between the two cutoffs) and deletes nothing. `GET /v1/admin/retention/plan?days=N` returns `would_delete` by store and `held` apart, and `sb tenant retention --plan [--days N]` prints it. `TestLAW16_RetentionPlanIsWhatTheSweepDeletes` runs the plan and then the sweep over random worlds with holds and shorter spaces and requires the same counts, with self-tests for a plan that ignores holds, one that deletes and one with its own cutoff; the Postgres test checks plan against expire store by store.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
