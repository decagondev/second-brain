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
| Status | In review |

## Why

Retention is configuration, not a project.

## Acceptance criteria

- [ ] `retention_days` enforced on derived data and logs; audit retention follows its own policy (the job and every store are built and tested; running it nightly at 03:00 UTC needs the cell runtime's scheduler, W7)
- [x] Subject deletion request removes derived data and records the deletion in audit

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retention jobs
- [x] Deletion API
- [x] Tests
- [x] Write or extend the law tests: LAW-13
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.3c): `internal/core/retention.Job` is the nightly sweep and subject erasure. It runs over the cell's stores of derived data only: traces (answers and drafts, with their citations and decision lists), conversations, feedback, and the daily log files (`internal/adapters/logs/localfs`). Audit and the index are never among them. A record exactly `retention_days` old is kept. A `retention_days` outside 30 to 3650 refuses to run. Each sweep writes one `kind: retention` audit row with the cutoff and the count per store, and a failing store is recorded and retried the next night. `POST /v1/admin/subjects/erase` (operator only) audits the request before erasing anything: if that row can't be written, nothing is erased. It then erases the user's records from every store, log lines that name them included, and audits the outcome. `retention` is a new audit kind (sb-audit(5)). The new law test, `TestLAW13_RetentionLeavesTheIndexDerived`, checks the sweep and an erasure against a model and shows that the index still equals a rebuild. A self-test shows that a job which ages the index, and one which deletes records exactly `retention_days` old, are both caught.

**Left:** the nightly schedule and the Postgres stores in a running cell (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
