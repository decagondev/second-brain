---
title: "SB-901: Cloud Run services and jobs per cell"
summary: "A cell needs to run somewhere real; Cloud Run gives a managed runtime without Kubernetes (ADR-0011)."
---

# SB-901: Cloud Run services and jobs per cell

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.1 Run a cell |
| Stories | US-09.1 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md), `sb-runtime(7)` (new) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-703](../E08-factory-handoff/SB-703-placement-adapter-for-tenant-owned-cloud.md) |
| Status | Open |

## Why

A cell needs to run somewhere real; Cloud Run gives a managed runtime without Kubernetes (ADR-0011).

## Acceptance criteria

- [ ] `api`, `ingest` and `web` services and the job runner deploy into the cell's project, each with its own service account and least privilege
- [ ] Only images the release controller verified are admitted (Binary Authorization), and an unsigned image is refused (LAW-06)
- [ ] The deploy is pulled by the cell, never pushed by the factory (ADR-0015)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `sb-runtime(7)` first
- [ ] Runtime port and Cloud Run adapter
- [ ] Admission policy
- [ ] Contract test against a recorded fake
- [ ] Write or extend the law tests: LAW-02, LAW-06
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
