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
| Status | Open |

## Why

The manifest is the truth; drift is a defect.

## Acceptance criteria

- [ ] Hourly comparison of live cell config with the applied manifest; drift alert names the field

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Live config reader
- [ ] Diff
- [ ] Alert
- [ ] Write or extend the law tests: LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
