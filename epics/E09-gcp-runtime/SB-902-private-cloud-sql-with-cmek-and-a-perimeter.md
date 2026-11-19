---
title: "SB-902: Private Cloud SQL with CMEK and a VPC Service Controls perimeter"
summary: "The database holds the index, audit and approvals; it must be reachable only from its own cell and encrypted under the cell's key."
---

# SB-902: Private Cloud SQL with CMEK and a VPC Service Controls perimeter

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.1 Run a cell |
| Stories | US-09.1 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md), `sb-runtime(7)` (new) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-901](SB-901-cloud-run-services-per-cell.md) |
| Status | Open |

## Why

The database holds the index, audit and approvals; it must be reachable only from its own cell and encrypted under the cell's key.

## Acceptance criteria

- [ ] Cloud SQL for PostgreSQL 17 with `pgvector`, private IP only, reached by Direct VPC egress
- [ ] Encrypted with a CMEK from the cell's key ring; destroying the key makes the data unreadable (tested in the fake)
- [ ] A VPC Service Controls perimeter around the project; a call from outside it is refused

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Network and connection
- [ ] CMEK binding
- [ ] Perimeter
- [ ] LAW-02 test over two perimeters
- [ ] Write or extend the law tests: LAW-02, LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
