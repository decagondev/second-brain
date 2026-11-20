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
| Status | In review |

## Why

The database holds the index, audit and approvals; it must be reachable only from its own cell and encrypted under the cell's key.

## Acceptance criteria

- [ ] Cloud SQL for PostgreSQL 17 with `pgvector`, private IP only, reached by Direct VPC egress (private IP and the flag in the adapter and fake; the VPC and egress in a real project are SB-903)
- [x] Encrypted with a CMEK from the cell's key ring; destroying the key makes the data unreadable (tested in the fake)
- [ ] A VPC Service Controls perimeter around the project; a call from outside it is refused (in the fake, and the bootstrap script; a real org is SB-903)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Network and connection
- [x] CMEK binding
- [x] Perimeter
- [x] LAW-02 test over two perimeters
- [x] Write or extend the law tests: LAW-02, LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S9.1b): sb-runtime(7) "Data, exactly" and sb-tenancy(7) say how the database is made. The gcp adapter now makes the cell's Cloud SQL instance with private IP only on the cell's network, the `pgvector` flag, and a CMEK key `cell-TENANT-sql` in the cell's own key ring. It grants that key only to the project's own Cloud SQL service agent, looking up the project number to name it. An existing instance that's public or on another key is refused. The deploy role gained `resourcemanager.projects.get` and `cloudkms.cryptoKeys.setIamPolicy`, in code and in the bootstrap script, and the every-permission-is-needed test still holds with 16. `deploy/gcp/bootstrap-perimeter.sh` creates the VPC Service Controls perimeter. The GCP fake now checks CMEK keys and their agent, refuses reads once a key is destroyed, and enforces perimeters by principal. `TestDatabaseIsPrivateWithCMEK` shows crypto-shredding, and `TestPublicInstanceRefused` shows a public instance isn't reused. The new law test `TestLAW02_PerimetersHoldEvenWithAnIAMMistake` shows that with a wrong cross-cell IAM grant, one cell's identity still can't read, decrypt or list the other's data. A self-test catches a perimeter that lets the other cell in.

**Left:** the VPC, Direct VPC egress and a real perimeter in a real project and organisation (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
