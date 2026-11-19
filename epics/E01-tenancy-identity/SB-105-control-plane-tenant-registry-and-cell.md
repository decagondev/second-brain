---
title: "SB-105: Control-plane tenant registry and cell routing"
summary: "Requests must find their cell without any code knowing a company's name."
---

# SB-105: Control-plane tenant registry and cell routing

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.2 Cell |
| Stories | US-01.1 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-103](../E01-tenancy-identity/SB-103-cell-template-infrastructure-as-code.md) |
| Status | In review |

## Why

Requests must find their cell without any code knowing a company's name.

## Acceptance criteria

- [x] Registry maps tenant id to cell endpoint and region
- [x] Router refuses requests whose token tenant does not match the cell
- [x] Registry stores the custody record (`custody`, `placement`, `key_custodian`) for every cell; v1 writes `factory` only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Registry table and API
- [x] Edge router with tenant check
- [x] Cross-tenant request test returns `ETENANT`
- [x] Cell holds a control-plane endpoint and identity certificate as config, so it can be re-pointed
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built.
- The new ports are `Cells` and `CellsWriter` (`CellRecord`: tenant, region, endpoint, hosts and the custody record).
- `internal/adapters/postgres/registry` (pgx, ADR-0017; first vendored module) implements the whole registry: applied resources, cells, hosts and journal. The `cell_hosts` primary key means a host can serve only one cell, and `PutCell` is a single transaction.
- `localfs` implements the same ports for `file:///DIR`. Both pass `internal/core/registrytest.Contract`, and Postgres runs it against a real database (`make integration`, new CI job `postgres`).
- `sb tenant apply` writes the cell record (custody `factory`, placement `factory-account`) when it creates a cell.
- `internal/edge` routes in the order of sb-tenancy(7): unknown host is `EPERM`, registry down is `EUNAVAIL`, a bad token is `EAUTH`, and another tenant's token is `ETENANT`. It forwards only to the host's cell.
- `cell.LoadConfig` takes the control-plane URL and identity certificate from configuration, with no defaults.

LAW-02 gains a request-level law test, which was red first, then green. On its first green run it caught a spec gap: a valid token with no tenant claim. That is now `EAUTH` in the manual and in the law.

**Left:** token verification is a placeholder port until SB-106 (OIDC). Also left: review by the user, and the first green CI run, including the `postgres` job.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
