---
title: "SB-1506: sb tenant apply against a running cell"
summary: "Apply plans and checks offline; a running cell must take the manifest and match it."
---

# SB-1506: `sb tenant apply` against a running cell

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.3 |
| Stories | US-15.1 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | LAW-16 |
| Depends on | [SB-1502](SB-1502-api-service.md) |
| Status | Done |

## Why

Apply plans and checks offline; a running cell must take the manifest and match it.

## Acceptance criteria

- [x] Apply sends the checked manifest to the cell, which syncs spaces (refusing a held space), groups, workflows and duties, and audits the change
- [x] Drift is reported hourly from the running cell (sb-tenant(5), Drift)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Apply endpoint and client
- [x] Space and group sync
- [x] Drift reporter
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.3): `wire.Cell.Apply` and `POST /v1/cell/manifest` (control plane only). The composition root is split: the stores and sign-in are built once, and everything that comes from the manifest is assembled into `wire.Parts` and swapped in whole by an apply, so a request sees one manifest. Apply re-checks the manifest, syncs the spaces registry (refusing with `EHOLD` and changing nothing if a held space would go), applies an Identity Platform cell's groups (now also at start, which was missing), rewrites the manifest file and writes a `config` audit row; the same manifest twice changes nothing. `sb tenant apply FILE --cell URL` sends it after the control plane's apply. Drift: the hourly `drift` job compares what the cell runs (with its registry's spaces) against the manifest file it last applied and sends `manifest_drift` per difference; `GET /v1/cell/live` gives the control plane the same resources. Tested in `tests/cell` (`TestApplyToARunningCell`, `TestApplyRefusesToRemoveAHeldSpace`, `TestDriftFromTheRunningCell`) and `internal/cli` (`TestTenantApplyCell`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
