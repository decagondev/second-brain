---
title: "SB-903: sb tenant apply in a real GCP project in under 30 minutes"
summary: "SB-104 and SB-703 are In review until a cell is stamped for real."
---

# SB-903: `sb tenant apply` in a real GCP project in under 30 minutes

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.1 Run a cell |
| Stories | US-09.1 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md), [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-902](SB-902-private-cloud-sql-with-cmek-and-a-perimeter.md), [SB-104](../E01-tenancy-identity/SB-104-sb-tenant-apply-idempotent-cell-creation.md) |
| Status | Open |

## Why

SB-104 and SB-703 are In review until a cell is stamped for real.

## Acceptance criteria

- [ ] A new cell in a real project, from an empty folder, in under 30 minutes, and a second apply changes nothing
- [ ] Two cells in two projects share nothing (LAW-02, run against real IAM)
- [ ] SB-104 and SB-703 move to Done

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Bootstrap the folder and deploy role
- [ ] Timed run
- [ ] Idempotence run
- [ ] Close SB-104 and SB-703
- [ ] Write or extend the law tests: LAW-02, LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
