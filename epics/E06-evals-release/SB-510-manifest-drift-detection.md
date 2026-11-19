---
title: "SB-510: Manifest drift detection"
summary: "The manifest is the truth; drift is a defect."
---

# SB-510: Manifest drift detection

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.3 Rings |
| Stories | US-06.4 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-104](../E01-tenancy-identity/SB-104-sb-tenant-apply-idempotent-cell-creation.md) |
| Status | In review |

## Why

The manifest is the truth; drift is a defect.

## Acceptance criteria

- [x] Hourly comparison of live cell config with the applied manifest; drift alert names the field

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Live config reader (cells report live resources in plan terms; the reader in a running cell comes with W7)
- [x] Diff
- [x] Alert
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.3b): `internal/core/drift.Detect` compares live resources with the applied ones and reports each changed attribute, and each resource present on one side only, with a message that names the resource and field (for example `connector tickets: attribute vendor: applied "jira", live "linear"`). It never changes anything: `sb tenant apply` is the fix. `TestLAW16_DriftIsReported` went red first, then green. It is a property over random applied states and live changes, and it requires every change to be reported and nothing else. A detector that only compares attributes is caught.

**Left:** the cell's live-config reporter and the hourly schedule (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
