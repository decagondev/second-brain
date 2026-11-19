---
title: "SB-114: Factory seams: custody record, placement port, key hierarchy"
summary: "Hand-off is planned for W7; these v1 seams make it a workflow later instead of a migration (ADR-0015)."
---

# SB-114: Factory seams: custody record, placement port, key hierarchy

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W1 / S1.2 Cell |
| Stories | US-08.1 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md) |
| Laws | [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-103](../E01-tenancy-identity/SB-103-cell-template-infrastructure-as-code.md), [SB-105](../E01-tenancy-identity/SB-105-control-plane-tenant-registry-and-cell.md) |
| Status | In review |

## Why

Hand-off is planned for W7; these v1 seams make it a workflow later instead of a migration (ADR-0015).

## Acceptance criteria

- [x] Custody fields exist in the registry and in `sb factory status` output (v1 values only)
- [x] A lint fails any code that reads an account id or key custodian from anywhere but the custody record
- [x] `sb tenant check` rejects `custody`, `placement` and `key_custodian` in a manifest with a pointer to `sb-factory(7)`
- [x] Design note in the devlog listing every seam and its test

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Add custody columns and reader API to the registry
- [x] Define the `Placement` port with a factory-account adapter only
- [x] Write the custody-field lint
- [x] Manifest rejection messages and golden tests
- [x] Read-only `sb factory status`
- [x] Write or extend the law tests: LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: most seams arrived with SB-101 (manifest rejections), SB-103 (`Placement` and `Keys` ports, envelope keys) and SB-105 (custody columns and the `CellByTenant` reader). This ticket adds:
- the custody constants, defined once in `internal/core/cell/custody.go`;
- `tools/check/custody.py` (in `make lint`, with a self-test), which refuses account addresses, 12-digit ids and custody literals anywhere else;
- read-only `sb factory status TENANT`, with a smoke test;
- the first LAW-18 law test: every recorded cell names a key custodian, and its root key unwraps its data keys.

There was no red phase for the law test, because the code it tests already existed. Instead, its self-test shows that a missing custodian and a dead root key are both caught. The seams are listed in the devlog design note.

**Left:** review by the user, (first CI run green on `c0a12be` (all jobs: check, proofs, postgres, diagrams)). Hand-off itself is W7.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
