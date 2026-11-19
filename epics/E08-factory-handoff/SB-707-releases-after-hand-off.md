---
title: "SB-707: Releases after hand-off"
summary: "A handed-off cell stays patched by pulling from the train."
---

# SB-707: Releases after hand-off

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.4 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-08](../../manual/sb-laws.7.md#law-08) |
| Depends on | [SB-705](../E08-factory-handoff/SB-705-hand-off-workflow-with-dual-approval.md), [SB-507](../E06-evals-release/SB-507-release-controller-with-channels-and-rings.md) |
| Status | Open |

## Why

A handed-off cell stays patched by pulling from the train.

## Acceptance criteria

- [ ] Tenant-custody cell pulls and verifies bundles with only the `publish` grant
- [ ] Promote guard counts tenant-custody cells

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Pull path test from a tenant account
- [ ] Controller includes tenant-custody cells
- [ ] Runbook for a tenant pausing pulls (becomes detach)
- [ ] Write or extend the law tests: LAW-06, LAW-08
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
